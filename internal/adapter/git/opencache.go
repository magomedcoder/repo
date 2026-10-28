package git

import (
	"sync"

	gogit "github.com/go-git/go-git/v6"
	"repo.magomedcoder.ru/repo/internal/domain"
)

type openCache struct {
	mu      sync.Mutex
	max     int
	entries map[string]*gogit.Repository
	order   []string
}

func newOpenCache(max int) *openCache {
	if max <= 0 {
		max = 32
	}

	return &openCache{
		max:     max,
		entries: make(map[string]*gogit.Repository),
	}
}

func (c *openCache) get(path string) (*gogit.Repository, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	repo, ok := c.entries[path]
	return repo, ok
}

func (c *openCache) put(path string, repo *gogit.Repository) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.entries[path]; exists {
		return
	}

	c.entries[path] = repo
	c.order = append(c.order, path)
	for len(c.order) > c.max {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.entries, oldest)
	}
}

func (c *openCache) evict(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.entries[path]; !ok {
		return
	}

	delete(c.entries, path)
	for i, p := range c.order {
		if p == path {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
}

type statsCache struct {
	mu      sync.Mutex
	entries map[string]*domain.RepoStats
}

func newStatsCache() *statsCache {
	return &statsCache{entries: make(map[string]*domain.RepoStats)}
}

func (c *statsCache) get(key string) (*domain.RepoStats, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	stats, ok := c.entries[key]
	return stats, ok
}

func (c *statsCache) put(key string, stats *domain.RepoStats) {
	if stats == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	cp := *stats
	if stats.Languages != nil {
		cp.Languages = make(map[string]int64, len(stats.Languages))
		for k, v := range stats.Languages {
			cp.Languages[k] = v
		}
	}
	c.entries[key] = &cp
}

func (c *statsCache) evictPrefix(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.entries {
		if len(key) >= len(path) && key[:len(path)] == path {
			delete(c.entries, key)
		}
	}
}
