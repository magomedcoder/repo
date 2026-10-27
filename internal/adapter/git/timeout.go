package git

import (
	"context"
	"fmt"
	"time"
)

const gitOpTimeout = 30 * time.Second

func withGitTimeout[T any](fn func() (T, error)) (T, error) {
	var zero T
	ctx, cancel := context.WithTimeout(context.Background(), gitOpTimeout)
	defer cancel()

	type result struct {
		v   T
		err error
	}
	ch := make(chan result, 1)
	go func() {
		v, err := fn()
		ch <- result{v: v, err: err}
	}()

	select {
	case res := <-ch:
		return res.v, res.err
	case <-ctx.Done():
		return zero, fmt.Errorf("git operation timed out")
	}
}
