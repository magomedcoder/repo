package syncutil

import (
	"context"
	"errors"
	"time"
)

var ErrBusy = errors.New("busy")

func Acquire(ctx context.Context, sem chan struct{}, wait time.Duration) (func(), error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	select {
	case sem <- struct{}{}:
		return func() { <-sem }, nil
	default:
	}

	t := time.NewTimer(wait)
	defer t.Stop()

	select {
	case sem <- struct{}{}:
		return func() { <-sem }, nil
	case <-t.C:
		return nil, ErrBusy
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
