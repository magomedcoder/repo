package git

import (
	"maps"

	gogit "github.com/go-git/go-git/v6"
	"github.com/magomedcoder/repo/internal/domain"
	"github.com/magomedcoder/repo/pkg/lrucache"
)

type openCache struct {
	c *lrucache.Cache[*gogit.Repository]
}

func newOpenCache(max int) *openCache {
	return &openCache{
		c: lrucache.New(max, func(repo *gogit.Repository) {
			if repo != nil {
				_ = repo.Close()
			}
		}),
	}
}

func (c *openCache) getOrOpen(path string, open func() (*gogit.Repository, error)) (*gogit.Repository, error) {
	return c.c.GetOrOpen(path, open)
}

func (c *openCache) Len() int {
	return c.c.Len()
}

func (c *openCache) evict(path string) {
	c.c.Evict(path)
}

type statsCache struct {
	c *lrucache.Cache[*domain.RepoStats]
}

func newStatsCache() *statsCache {
	return &statsCache{
		c: lrucache.New[*domain.RepoStats](256, nil),
	}
}

func (c *statsCache) get(key string) (*domain.RepoStats, bool) {
	return c.c.Get(key)
}

func (c *statsCache) put(key string, stats *domain.RepoStats) {
	if stats == nil {
		return
	}
	cp := *stats
	if stats.Languages != nil {
		cp.Languages = maps.Clone(stats.Languages)
	}
	c.c.Put(key, &cp)
}

func (c *statsCache) evictPrefix(path string) {
	c.c.EvictPrefix(path)
}
