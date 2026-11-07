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

	if !c.c.Has("a") {
		t.Fatal("a evicted despite recent hit")
	}
	if c.c.Has("b") {
		t.Fatal("b should have been evicted")
	}
}
