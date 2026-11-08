package argon2id

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestHashCompareRoundTrip(t *testing.T) {
	h := NewHasher()
	hash, err := h.Hash("correct horse")
	if err != nil {
		t.Fatal(err)
	}

	if err := h.Compare(hash, "correct horse"); err != nil {
		t.Fatalf("compare: %v", err)
	}

	if err := h.Compare(hash, "wrong"); err == nil {
		t.Fatal("wrong password must not match")
	}

	if h.NeedsRehash(hash) {
		t.Fatal("fresh hash must not need rehash")
	}

	old := strings.Replace(hash, "t=3", "t=2", 1)
	if !h.NeedsRehash(old) {
		t.Fatal("t=2 hash must be upgraded to the current time cost")
	}
}

func TestRejectsOversizedPassword(t *testing.T) {
	h := NewHasher()
	big := strings.Repeat("x", MaxPasswordLen+1)

	if _, err := h.Hash(big); err == nil {
		t.Fatal("Hash must reject oversized passwords")
	}

	if err := h.Compare("argon2id$v=19$m=65536,t=3,p=4$c2FsdA$AAAA", big); err == nil {
		t.Fatal("Compare must reject oversized passwords")
	}
}

func TestSemaphoreReturnsBusy(t *testing.T) {
	SetConcurrency(1)
	SetAcquireTimeout(30 * time.Millisecond)
	t.Cleanup(func() {
		SetConcurrency(defaultConcurrency())
		SetAcquireTimeout(defaultAcquireTimeout)
	})

	release, err := acquire()
	if err != nil {
		t.Fatal(err)
	}

	h := NewHasher()
	if _, err := h.Hash("whatever-password"); !errors.Is(err, ErrBusy) {
		t.Fatalf("Hash with saturated semaphore: %v", err)
	}

	if err := h.Compare("argon2id$v=19$m=65536,t=3,p=4$c2FsdHNhbHRzYWx0c2FsdA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", "x"); !errors.Is(err, ErrBusy) {
		t.Fatalf("Compare with saturated semaphore: %v", err)
	}

	release()
	if _, err := h.Hash("whatever-password"); err != nil {
		t.Fatalf("slot freed: %v", err)
	}
}
