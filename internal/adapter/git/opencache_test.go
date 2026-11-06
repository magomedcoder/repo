package git

import (
	"testing"

	gogit "github.com/go-git/go-git/v6"
)

func TestOpenCacheLRUOnHit(t *testing.T) {
	t.Parallel()
	c := newOpenCache(2)
	open := func() (*gogit.Repository, error) {
		return &gogit.Repository{}, nil
	}

	for _, p := range []string{"a", "b"} {
		if _, err := c.getOrOpen(p, open); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := c.getOrOpen("a", open); err != nil {
		t.Fatal(err)
	}

	if _, err := c.getOrOpen("c", open); err != nil {
		t.Fatal(err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.entries["a"]; !ok {
		t.Fatal("a evicted despite recent hit")
	}

	if _, ok := c.entries["b"]; ok {
		t.Fatal("b should have been evicted")
	}
}
