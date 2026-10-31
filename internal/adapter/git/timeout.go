package git

import (
	"context"
	"time"
)

func (r *Repository) withTimeout[T any](ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	return fn(ctx)
}
