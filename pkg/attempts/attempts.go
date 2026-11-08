package attempts

import (
	"context"
	"strings"
	"sync"
	"time"
)

const (
	defaultMaxFails     = 5
	defaultLockDuration = 15 * time.Minute
	sweepEvery          = 64
	idleTTL             = 30 * time.Minute
	defaultMaxKeys      = 8192
	maxKeysPerScope     = 64
)

func scopeOf(key string) string {
	parts := strings.SplitN(key, "\n", 3)
	if len(parts) < 3 {
		return ""
	}

	switch parts[0] {
	case "login-user", "git":
		return parts[0] + "\n" + parts[1]
	}

	return ""
}

type Counter interface {
	Fail(key string) bool
	Reset(key string)
	Locked(key string) bool
	RetryAfter(key string) time.Duration
	Sweep()
}

func StartBackgroundSweep(ctx context.Context, every time.Duration, counters ...Counter) {
	if every <= 0 {
		every = 5 * time.Minute
	}

	go func() {
		t := time.NewTicker(every)
		defer t.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				for _, c := range counters {
					if c != nil {
						c.Sweep()
					}
				}
			}
		}
	}()
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
	lastFail    time.Time
}

func (st *state) quietSince() time.Time {
	if st.lockedUntil.After(st.lastFail) {
		return st.lockedUntil
	}

	return st.lastFail
}

func (st *state) locked(now time.Time) bool {
	return !st.lockedUntil.IsZero() && now.Before(st.lockedUntil)
}

func (st *state) decay(now time.Time) {
	if st.locked(now) {
		return
	}

	if q := st.quietSince(); !q.IsZero() && now.Sub(q) > idleTTL {
		st.fails = 0
		st.lockLevel = 0
		st.lockedUntil = time.Time{}
	}
}

type memory struct {
	mu        sync.Mutex
	fails     map[string]*state
	scopes    map[string]int
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
		scopes:    map[string]int{},
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
	m.maybeSweep(now)

	st := m.fails[key]
	if st == nil {
		if len(m.fails) >= m.maxKeys {
			m.sweep(now)
			m.lastSweep = now
			m.evictOldest(now)
		}
		if scope := scopeOf(key); scope != "" {
			if m.scopes[scope] >= maxKeysPerScope && !m.evictOldestInScope(scope, now) {
				return true
			}

			m.scopes[scope]++
		}
		st = &state{}
		m.fails[key] = st
	}
	st.lastUsed = now

	if st.locked(now) {
		return true
	}

	st.decay(now)
	if !st.lockedUntil.IsZero() {
		st.fails = 0
		st.lockedUntil = time.Time{}
	}

	st.lastFail = now
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
	return m.RetryAfter(key) > 0
}

func (m *memory) RetryAfter(key string) time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.now()
	m.maybeSweep(now)

	st := m.fails[key]
	if st == nil {
		return 0
	}

	st.lastUsed = now

	if st.lockedUntil.IsZero() || !now.Before(st.lockedUntil) {
		return 0
	}

	return st.lockedUntil.Sub(now)
}

func (m *memory) Reset(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleteKey(key)
}

func (m *memory) deleteKey(key string) {
	if _, ok := m.fails[key]; !ok {
		return
	}

	delete(m.fails, key)
	if scope := scopeOf(key); scope != "" {
		if m.scopes[scope] <= 1 {
			delete(m.scopes, scope)
		} else {
			m.scopes[scope]--
		}
	}
}

func (m *memory) evictOldestInScope(scope string, now time.Time) bool {
	var (
		oldestKey string
		oldest    time.Time
		found     bool
	)
	for k, st := range m.fails {
		if scopeOf(k) != scope {
			continue
		}

		if st.locked(now) {
			continue
		}

		if !found || st.lastUsed.Before(oldest) {
			oldestKey, oldest, found = k, st.lastUsed, true
		}
	}

	if found {
		m.deleteKey(oldestKey)
	}

	return found
}

func (m *memory) Sweep() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.now()
	m.sweep(now)
	m.lastSweep = now
	m.calls = 0
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
		if st.locked(now) {
			continue
		}

		st.decay(now)
		q := st.quietSince()
		if q.IsZero() {
			q = st.lastUsed
		}

		if !q.IsZero() && now.Sub(q) > idleTTL {
			m.deleteKey(k)
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
		if st.locked(now) {
			continue
		}

		if !found || st.lastUsed.Before(oldest) {
			oldestKey, oldest, found = k, st.lastUsed, true
		}
	}

	if found {
		m.deleteKey(oldestKey)
	}
}
