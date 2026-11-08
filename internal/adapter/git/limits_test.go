package git

import (
	"context"
	"errors"
	"testing"
	"time"

	"magomedcoder.ru/repo/internal/domain"
)

func TestValidateRefName(t *testing.T) {
	t.Parallel()

	ok := []string{"", "main", "feature/x-1", "v1.0.0", "refs/heads/main", "user@feat", "0123456789abcdef0123"}
	for _, ref := range ok {
		if err := validateRefName(ref); err != nil {
			t.Errorf("%q rejected: %v", ref, err)
		}
	}

	bad := []string{"main^{tree}", "a b", "HEAD~2", "a..b", "-x", "a:b", "x@{1}", "a\x00b", "a//b", "x.lock", "/x", "x/", "a*b", "a?b", "a[0]", "a\\b"}
	for _, ref := range bad {
		if err := validateRefName(ref); err == nil {
			t.Errorf("%q accepted", ref)
		}
	}
}

func TestLockRepoSerializes(t *testing.T) {
	t.Parallel()

	r := NewRepository(t.TempDir())
	p := r.Root + "/a.git"
	key := r.lockKey(p)
	unlock := r.LockRepo(p)
	got := make(chan struct{})
	go func() {
		defer close(got)
		r.LockRepo(p)()
	}()

	select {
	case <-got:
		t.Fatal("second lock acquired while held")
	default:
	}

	unlock()
	<-got
	if repoLocks.Has(key) {
		t.Fatal("lock entry leaked")
	}
}

func TestAcquireSemRespectsContext(t *testing.T) {
	t.Parallel()

	sem := make(chan struct{}, 1)
	release, err := acquireSem(context.Background(), sem)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := acquireSem(ctx, sem); !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}

	release()
	if r, err := acquireSem(context.Background(), sem); err != nil {
		t.Fatal(err)
	} else {
		r()
	}
}

func TestMoveAndRemoveEvictOpenCache(t *testing.T) {
	t.Parallel()

	r := NewRepository(t.TempDir())
	oldPath := r.Root + "/old.git"
	newPath := r.Root + "/new.git"
	if err := r.InitBare(oldPath, domain.BareInitOptions{}); err != nil {
		t.Fatal(err)
	}

	if _, err := r.open(oldPath); err != nil {
		t.Fatal(err)
	}

	if n := r.OpenCacheLen(); n != 1 {
		t.Fatalf("cache len=%d want 1", n)
	}

	if err := r.Move(oldPath, newPath); err != nil {
		t.Fatal(err)
	}

	if n := r.OpenCacheLen(); n != 0 {
		t.Fatalf("cache len after Move=%d want 0", n)
	}

	if _, err := r.open(newPath); err != nil {
		t.Fatal(err)
	}

	if err := r.Remove(newPath); err != nil {
		t.Fatal(err)
	}

	if n := r.OpenCacheLen(); n != 0 {
		t.Fatalf("cache len after Remove=%d want 0", n)
	}
}

func TestDiffSemBusy(t *testing.T) {
	for range cap(diffSem) {
		diffSem <- struct{}{}
	}

	defer func() {
		for range cap(diffSem) {
			<-diffSem
		}
	}()

	if _, err := acquireSemWithTimeout(context.Background(), diffSem, 20*time.Millisecond); !errors.Is(err, domain.ErrBusy) {
		t.Fatalf("err=%v want ErrBusy", err)
	}
}
