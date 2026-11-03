package attempts

import (
	"sync"
	"time"
)

const (
	defaultMaxFails     = 5
	defaultLockDuration = 15 * time.Minute
	sweepEvery          = 64
	idleTTL             = 30 * time.Minute
	defaultMaxKeys      = 100_000
)

type Counter interface {
	Fail(key string) bool
	Reset(key string)
	Locked(key string) bool
}

type Config struct {
	MaxFails int
	BaseLock time.Duration
	MaxKeys  int
	Now      func() time.Time
}

type state struct {
	fails       int
	lockedUntil time.Time
	lockLevel   int
	lastUsed    time.Time
}

type memory struct {
	mu        sync.Mutex
	fails     map[string]*state
	now       func() time.Time
	maxFail   int
	maxKeys   int
	base      time.Duration
	calls     int
	lastSweep time.Time
}

func New(cfg ...Config) Counter {
	c := Config{}
	if len(cfg) > 0 {
		c = cfg[0]
	}

	if c.MaxFails <= 0 {
		c.MaxFails = defaultMaxFails
	}

	if c.BaseLock <= 0 {
		c.BaseLock = defaultLockDuration
	}

	if c.MaxKeys <= 0 {
		c.MaxKeys = defaultMaxKeys
	}

	if c.Now == nil {
		c.Now = time.Now
	}

	now := c.Now()
	return &memory{
		fails:     map[string]*state{},
		now:       c.Now,
		maxFail:   c.MaxFails,
		maxKeys:   c.MaxKeys,
		base:      c.BaseLock,
		lastSweep: now,
	}
}

func (m *memory) Fail(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.now()
	st := m.fails[key]
	if st == nil {
		if len(m.fails) >= m.maxKeys {
			m.sweep(now)
			m.lastSweep = now
			m.evictOldest(now)
		}
		st = &state{}
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

func (m *memory) Locked(key string) bool {
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

func (m *memory) Reset(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.fails, key)
}

func (m *memory) maybeSweep(now time.Time) {
	m.calls++
	if m.calls < sweepEvery && now.Sub(m.lastSweep) <= 10*time.Minute {
		return
	}

	m.sweep(now)
	m.lastSweep = now
	m.calls = 0
}

func (m *memory) sweep(now time.Time) {
	for k, st := range m.fails {
		lockExpired := !st.lockedUntil.IsZero() && !now.Before(st.lockedUntil) && st.fails == 0
		idle := !st.lastUsed.IsZero() && now.Sub(st.lastUsed) > idleTTL
		if lockExpired || idle {
			delete(m.fails, k)
		}
	}
}

func (m *memory) evictOldest(now time.Time) {
	if len(m.fails) < m.maxKeys {
		return
	}

	var (
		oldestKey string
		oldest    time.Time
		found     bool
	)
	for k, st := range m.fails {
		if !st.lockedUntil.IsZero() && now.Before(st.lockedUntil) {
			continue
		}

		if !found || st.lastUsed.Before(oldest) {
			oldestKey, oldest, found = k, st.lastUsed, true
		}
	}

	if found {
		delete(m.fails, oldestKey)
	}
}
