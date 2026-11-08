package syncutil

import (
	"slices"
	"sync"
)

type keyedEntry struct {
	mu   sync.Mutex
	refs int
}

type KeyedMutex struct {
	mu sync.Mutex
	m  map[string]*keyedEntry
}

func (k *KeyedMutex) Lock(key string) func() {
	k.mu.Lock()
	if k.m == nil {
		k.m = map[string]*keyedEntry{}
	}

	e := k.m[key]
	if e == nil {
		e = &keyedEntry{}
		k.m[key] = e
	}

	e.refs++
	k.mu.Unlock()

	e.mu.Lock()

	return func() {
		e.mu.Unlock()
		k.mu.Lock()
		e.refs--
		if e.refs == 0 {
			delete(k.m, key)
		}
		k.mu.Unlock()
	}
}

func (k *KeyedMutex) Len() int {
	k.mu.Lock()
	defer k.mu.Unlock()

	return len(k.m)
}

func (k *KeyedMutex) Has(key string) bool {
	k.mu.Lock()
	defer k.mu.Unlock()

	_, ok := k.m[key]
	return ok
}

func (k *KeyedMutex) LockAll(keys ...string) func() {
	uniq := make([]string, 0, len(keys))
	seen := map[string]bool{}
	for _, key := range keys {
		if !seen[key] {
			seen[key] = true
			uniq = append(uniq, key)
		}
	}

	slices.Sort(uniq)
	unlocks := make([]func(), 0, len(uniq))
	for _, key := range uniq {
		unlocks = append(unlocks, k.Lock(key))
	}

	return func() {
		for _, unlock := range slices.Backward(unlocks) {
			unlock()
		}
	}
}
