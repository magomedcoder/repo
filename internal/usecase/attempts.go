package usecase

import "sync"

const maxLoginFails = 5

type AttemptCounter interface {
	Fail(key string) bool
	Reset(key string)
	Locked(key string) bool
}

type memoryAttempts struct {
	mu    sync.Mutex
	fails map[string]int
}

func NewAttemptCounter() AttemptCounter {
	return &memoryAttempts{fails: map[string]int{}}
}

func (m *memoryAttempts) Fail(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fails[key]++
	return m.fails[key] > maxLoginFails
}

func (m *memoryAttempts) Locked(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.fails[key] >= maxLoginFails
}

func (m *memoryAttempts) Reset(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.fails, key)
}
