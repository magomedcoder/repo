package attempts

import (
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

func TestSweepDropsIdleKeys(t *testing.T) {
	t.Parallel()
	clk := &fakeClock{t: time.Unix(1_000_000, 0)}
	c := New(Config{Now: clk.now})
	m := c.(*memory)

	c.Fail("a")
	c.Fail("b")
	clk.advance(idleTTL + time.Minute)
	c.Sweep()

	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.fails) != 0 {
		t.Fatalf("idle keys not swept: %d", len(m.fails))
	}
}

func TestStartBackgroundSweep(t *testing.T) {
	t.Parallel()
	clk := &fakeClock{t: time.Unix(1_000_000, 0)}
	c := New(Config{Now: clk.now})
	m := c.(*memory)
	c.Fail("a")
	clk.advance(idleTTL + time.Minute)

	ctx := t.Context()
	StartBackgroundSweep(ctx, 5*time.Millisecond, c)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		n := len(m.fails)
		m.mu.Unlock()
		if n == 0 {
			return
		}

		time.Sleep(5 * time.Millisecond)
	}

	t.Fatal("background sweep did not clear idle key")
}

func TestDefaultMaxKeysLowered(t *testing.T) {
	t.Parallel()
	if defaultMaxKeys > 8192 {
		t.Fatalf("defaultMaxKeys=%d", defaultMaxKeys)
	}
}
