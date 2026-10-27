package usecase

import (
	"sync"
	"time"
)

const (
	maxLoginFails     = 5
	loginLockDuration = 15 * time.Minute
	attemptSweepEvery = 64
	attemptIdleTTL    = 30 * time.Minute
)

type AttemptCounter interface {
	Fail(key string) bool
	Reset(key string)
	Locked(key string) bool
}

type attemptState struct {
	fails       int
	lockedUntil time.Time
	lockLevel   int
	lastUsed    time.Time
}

type memoryAttempts struct {
	mu        sync.Mutex
	fails     map[string]*attemptState
	now       func() time.Time
	maxFail   int
	base      time.Duration
	calls     int
	lastSweep time.Time
}

func NewAttemptCounter() AttemptCounter {
	now := time.Now()
	return &memoryAttempts{
		fails:     map[string]*attemptState{},
		now:       time.Now,
		maxFail:   maxLoginFails,
		base:      loginLockDuration,
		lastSweep: now,
	}
}

func (m *memoryAttempts) Fail(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.now()
	st := m.fails[key]
	if st == nil {
		st = &attemptState{}
		m.fails[key] = st
	}
	st.lastUsed = now
	m.maybeSweep(now)

	if !st.lockedUntil.IsZero() && now.Before(st.lockedUntil) {
		return true
	}

	if !st.lockedUntil.IsZero() && !now.Before(st.lockedUntil) {
		st.fails = 0
		st.lockedUntil = time.Time{}
	}

	st.fails++
	if st.fails < m.maxFail {
		return false
	}

	st.lockLevel++
	mult := 1 << min(st.lockLevel-1, 3)
	st.lockedUntil = now.Add(m.base * time.Duration(mult))
	st.fails = 0
	return true
}

func (m *memoryAttempts) Locked(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	st := m.fails[key]
	if st == nil {
		return false
	}

	now := m.now()
	st.lastUsed = now
	m.maybeSweep(now)

	if st.lockedUntil.IsZero() {
		return false
	}

	if now.Before(st.lockedUntil) {
		return true
	}

	st.lockedUntil = time.Time{}
	return false
}

func (m *memoryAttempts) Reset(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.fails, key)
}

func (m *memoryAttempts) maybeSweep(now time.Time) {
	m.calls++
	if m.calls < attemptSweepEvery && now.Sub(m.lastSweep) <= 10*time.Minute {
		return
	}

	m.sweep(now)
	m.lastSweep = now
	m.calls = 0
}

func (m *memoryAttempts) sweep(now time.Time) {
	for k, st := range m.fails {
		lockExpired := !st.lockedUntil.IsZero() && !now.Before(st.lockedUntil) && st.fails == 0
		idle := !st.lastUsed.IsZero() && now.Sub(st.lastUsed) > attemptIdleTTL
		if lockExpired || idle {
			delete(m.fails, k)
		}
	}
}
