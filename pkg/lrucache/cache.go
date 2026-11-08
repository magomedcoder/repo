package lrucache

import (
	"errors"
	"sync"
)

var errOpenPanicked = errors.New("lrucache: open panicked")

type flight[V any] struct {
	done chan struct{}
	v    V
	err  error
}

type Cache[V any] struct {
	mu      sync.Mutex
	max     int
	entries map[string]V
	flights map[string]*flight[V]
	order   []string
	onEvict func(V)
}

func New[V any](max int, onEvict func(V)) *Cache[V] {
	if max <= 0 {
		max = 32
	}
	return &Cache[V]{
		max:     max,
		entries: make(map[string]V),
		flights: make(map[string]*flight[V]),
		onEvict: onEvict,
	}
}

func (c *Cache[V]) GetOrOpen(key string, open func() (V, error)) (V, error) {
	var zero V
	c.mu.Lock()
	if v, ok := c.entries[key]; ok {
		c.touchLocked(key)
		c.mu.Unlock()
		return v, nil
	}

	if f, ok := c.flights[key]; ok {
		c.mu.Unlock()
		<-f.done
		if f.err != nil {
			return zero, f.err
		}
		return f.v, nil
	}

	f := &flight[V]{done: make(chan struct{})}
	c.flights[key] = f
	c.mu.Unlock()

	var (
		v   V
		err = errOpenPanicked
	)
	defer func() {
		c.mu.Lock()
		delete(c.flights, key)
		if err == nil {
			v = c.storeLocked(key, v)
		}
		c.mu.Unlock()

		f.v, f.err = v, err
		close(f.done)
	}()

	v, err = open()
	if err != nil {
		return zero, err
	}

	return v, nil
}

func (c *Cache[V]) storeLocked(key string, v V) V {
	if existing, ok := c.entries[key]; ok {
		if c.onEvict != nil {
			c.onEvict(v)
		}
		c.touchLocked(key)
		return existing
	}

	c.entries[key] = v
	c.order = append(c.order, key)
	for len(c.order) > c.max {
		oldest := c.order[0]
		c.order = c.order[1:]
		if old, ok := c.entries[oldest]; ok {
			delete(c.entries, oldest)
			if c.onEvict != nil {
				c.onEvict(old)
			}
		}
	}

	return v
}

func (c *Cache[V]) Get(key string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.entries[key]
	return v, ok
}

func (c *Cache[V]) Has(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.entries[key]
	return ok
}

func (c *Cache[V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

func (c *Cache[V]) Put(key string, v V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.entries[key]; exists {
		c.entries[key] = v
		c.touchLocked(key)
		return
	}

	c.entries[key] = v
	c.order = append(c.order, key)

	for len(c.order) > c.max {
		oldest := c.order[0]
		c.order = c.order[1:]
		if old, ok := c.entries[oldest]; ok {
			delete(c.entries, oldest)
			if c.onEvict != nil {
				c.onEvict(old)
			}
		}
	}
}

func (c *Cache[V]) Evict(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	old, ok := c.entries[key]
	if !ok {
		return
	}
	delete(c.entries, key)

	if c.onEvict != nil {
		c.onEvict(old)
	}

	for i, p := range c.order {
		if p == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
}

func (c *Cache[V]) EvictPrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	kept := c.order[:0]
	for _, key := range c.order {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			if old, ok := c.entries[key]; ok {
				delete(c.entries, key)
				if c.onEvict != nil {
					c.onEvict(old)
				}
			}

			continue
		}
		kept = append(kept, key)
	}

	c.order = kept
}

func (c *Cache[V]) touchLocked(key string) {
	for i, p := range c.order {
		if p == key {
			c.order = append(append(c.order[:i], c.order[i+1:]...), key)
			return
		}
	}
}
