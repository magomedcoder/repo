package git

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"unicode"

	"magomedcoder.ru/repo/pkg/fsutil"
)

const (
	maxCompareCommits    = 200
	maxWalkCommits       = 200000
	maxComparePatchBytes = 8 << 20
	maxBlameFileBytes    = 2 << 20
	maxConcurrentBlames  = 4
	maxRefNameLen        = 255
)

var (
	ErrBlameTooLarge   = errors.New("file too large to blame")
	ErrHistoryTooLarge = errors.New("history too large")
)

var blameSem = make(chan struct{}, maxConcurrentBlames)

func validateRefName(ref string) error {
	if ref == "" {
		return nil
	}

	if len(ref) > maxRefNameLen {
		return fmt.Errorf("%w: invalid ref name", ErrRefNotFound)
	}

	if strings.Contains(ref, "..") || strings.Contains(ref, "@{") || strings.Contains(ref, "//") ||
		strings.HasPrefix(ref, "-") || strings.HasPrefix(ref, "/") || strings.HasSuffix(ref, "/") ||
		strings.HasSuffix(ref, ".") || strings.HasSuffix(ref, ".lock") {
		return fmt.Errorf("%w: invalid ref name", ErrRefNotFound)
	}

	for _, c := range ref {
		switch {
		case unicode.IsLetter(c), unicode.IsDigit(c):
		case strings.ContainsRune("-_./+@#=,", c):
		default:
			return fmt.Errorf("%w: invalid ref name", ErrRefNotFound)
		}
	}

	return nil
}

func checkCtx(ctx context.Context) error {
	if ctx == nil {
		return nil
	}

	return ctx.Err()
}

type repoLockEntry struct {
	mu   sync.Mutex
	refs int
}

var (
	repoLocksMu sync.Mutex
	repoLocks   = map[string]*repoLockEntry{}
)

func (r *Repository) lockKey(path string) string {
	if safe, err := fsutil.SafePath(r.Root, path); err == nil {
		return safe
	}

	return path
}

func (r *Repository) LockRepo(path string) func() {
	return lockRepoKey(r.lockKey(path))
}

func lockRepoKey(key string) func() {
	repoLocksMu.Lock()
	e := repoLocks[key]
	if e == nil {
		e = &repoLockEntry{}
		repoLocks[key] = e
	}

	e.refs++
	repoLocksMu.Unlock()

	e.mu.Lock()

	return func() {
		e.mu.Unlock()
		repoLocksMu.Lock()
		e.refs--
		if e.refs == 0 {
			delete(repoLocks, key)
		}
		repoLocksMu.Unlock()
	}
}

func (r *Repository) lockRepos(paths ...string) func() {
	keys := make([]string, 0, len(paths))
	seen := map[string]bool{}
	for _, p := range paths {
		k := r.lockKey(p)
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}

	slices.Sort(keys)
	unlocks := make([]func(), 0, len(keys))
	for _, k := range keys {
		unlocks = append(unlocks, lockRepoKey(k))
	}

	return func() {
		for _, unlock := range slices.Backward(unlocks) {
			unlock()
		}
	}
}
