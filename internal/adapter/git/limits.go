package git

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/magomedcoder/repo/internal/domain"
	"github.com/magomedcoder/repo/pkg/fsutil"
	"github.com/magomedcoder/repo/pkg/gitref"
	"github.com/magomedcoder/repo/pkg/syncutil"
)

const (
	maxCompareCommits    = 200
	maxWalkCommits       = 200000
	maxComparePatchBytes = 8 << 20
	maxBlameFileBytes    = 2 << 20
	maxConcurrentBlames  = 4
	maxConcurrentDiffs   = 8
)

var (
	ErrBlameTooLarge   = errors.New("file too large to blame")
	ErrHistoryTooLarge = errors.New("history too large")
)

const semAcquireTimeout = 5 * time.Second

var blameSem = make(chan struct{}, maxConcurrentBlames)

var diffSem = make(chan struct{}, maxConcurrentDiffs)

var repoLocks syncutil.KeyedMutex

func acquireSem(ctx context.Context, sem chan struct{}) (func(), error) {
	return acquireSemWithTimeout(ctx, sem, semAcquireTimeout)
}

func acquireSemWithTimeout(ctx context.Context, sem chan struct{}, wait time.Duration) (func(), error) {
	release, err := syncutil.Acquire(ctx, sem, wait)
	if errors.Is(err, syncutil.ErrBusy) {
		return nil, domain.ErrBusy
	}

	return release, err
}

func validateRefName(ref string) error {
	if err := gitref.Validate(ref); err != nil {
		return fmt.Errorf("%w: invalid ref name", ErrRefNotFound)
	}

	return nil
}

func checkCtx(ctx context.Context) error {
	if ctx == nil {
		return nil
	}

	return ctx.Err()
}

func (r *Repository) lockKey(path string) string {
	if safe, err := fsutil.SafePath(r.Root, path); err == nil {
		return safe
	}

	return path
}

func (r *Repository) LockRepo(path string) func() {
	return repoLocks.Lock(r.lockKey(path))
}

func (r *Repository) lockRepos(paths ...string) func() {
	keys := make([]string, 0, len(paths))
	for _, p := range paths {
		keys = append(keys, r.lockKey(p))
	}

	return repoLocks.LockAll(keys...)
}
