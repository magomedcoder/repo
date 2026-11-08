package lrucache

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetOrOpenSingleflight(t *testing.T) {
	t.Parallel()

	c := New[int](8, nil)
	var opens atomic.Int32
	release := make(chan struct{})

	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			v, err := c.GetOrOpen("k", func() (int, error) {
				opens.Add(1)
				<-release
				return 7, nil
			})
			if err != nil || v != 7 {
				t.Errorf("got %d, %v", v, err)
			}
		})
	}

	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	if n := opens.Load(); n != 1 {
		t.Fatalf("open called %d times, want 1", n)
	}
}

func TestGetOrOpenErrorNotCached(t *testing.T) {
	t.Parallel()

	c := New[int](8, nil)
	boom := errors.New("boom")
	if _, err := c.GetOrOpen("k", func() (int, error) {
		return 0, boom
	}); !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}

	v, err := c.GetOrOpen("k", func() (int, error) {
		return 3, nil
	})
	if err != nil || v != 3 {
		t.Fatalf("retry: %d %v", v, err)
	}
}

func TestGetOrOpenPanicReleasesFlight(t *testing.T) {
	t.Parallel()

	c := New[int](8, nil)
	func() {
		defer func() { _ = recover() }()
		_, _ = c.GetOrOpen("k", func() (int, error) {
			panic("x")
		})
	}()

	v, err := c.GetOrOpen("k", func() (int, error) {
		return 5, nil
	})
	if err != nil || v != 5 {
		t.Fatalf("after panic: %d %v", v, err)
	}
}
