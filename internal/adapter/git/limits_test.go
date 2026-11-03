package git

import "testing"

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
	repoLocksMu.Lock()
	_, leaked := repoLocks[r.lockKey(p)]
	repoLocksMu.Unlock()
	if leaked {
		t.Fatal("lock entry leaked")
	}
}
