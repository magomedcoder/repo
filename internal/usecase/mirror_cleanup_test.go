package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/magomedcoder/repo/internal/domain"
)

func mirrorRuntimeState(uc *RepositoryUseCase, id uint) (lock, debounce, pending bool) {
	uc.mirror.mu.Lock()
	defer uc.mirror.mu.Unlock()

	_, lock = uc.mirror.locks[id]
	_, debounce = uc.mirror.debounce[id]
	_, pending = uc.mirror.pending[id]

	return lock, debounce, pending
}

func seedMirrorRuntime(uc *RepositoryUseCase, id uint) {
	uc.mirrorLock(id)

	uc.mirror.mu.Lock()
	uc.mirror.debounce[id] = time.AfterFunc(time.Hour, func() {})
	uc.mirror.pending[id] = true
	uc.mirror.mu.Unlock()
}

func TestReleaseMirrorRuntime(t *testing.T) {
	t.Parallel()
	e := newMirrorEnv()
	seedMirrorRuntime(e.uc, 42)

	e.uc.releaseMirrorRuntime(42)

	if l, d, p := mirrorRuntimeState(e.uc, 42); l || d || p {
		t.Fatalf("runtime not released: lock=%v debounce=%v pending=%v", l, d, p)
	}
}

func TestDeleteReleasesMirrorRuntimeAndEvents(t *testing.T) {
	t.Parallel()
	e := newMirrorEnv()
	repo := e.addMirror(domain.MirrorPull)
	seedMirrorRuntime(e.uc, repo.ID)
	_ = e.events.Create(context.Background(), &domain.MirrorSyncEvent{
		RepoID: repo.ID,
	})

	if err := e.uc.Delete(context.Background(), DeleteRepositoryInput{
		Name:            "app",
		ViewerID:        1,
		Confirm:         "app",
		CurrentPassword: "secret",
	}); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if l, d, p := mirrorRuntimeState(e.uc, repo.ID); l || d || p {
		t.Fatalf("runtime leaked after delete: lock=%v debounce=%v pending=%v", l, d, p)
	}

	if evs, _ := e.events.ListByRepo(context.Background(), repo.ID, 10); len(evs) != 0 {
		t.Fatalf("mirror events not removed: %d", len(evs))
	}
}

func TestPurgeSoftDeletedReposClearsMirrorState(t *testing.T) {
	t.Parallel()
	e := newMirrorEnv()
	e.store.mu.Lock()
	e.store.purged = []uint{7, 8}
	e.store.mu.Unlock()

	for _, id := range []uint{7, 8} {
		seedMirrorRuntime(e.uc, id)
		_ = e.events.Create(context.Background(), &domain.MirrorSyncEvent{
			RepoID: id,
		})
	}

	n, err := e.uc.PurgeSoftDeletedRepos(context.Background(), 7*24*time.Hour)
	if err != nil || n != 2 {
		t.Fatalf("purge: n=%d err=%v", n, err)
	}

	for _, id := range []uint{7, 8} {
		if l, d, p := mirrorRuntimeState(e.uc, id); l || d || p {
			t.Fatalf("runtime leaked for %d", id)
		}

		if evs, _ := e.events.ListByRepo(context.Background(), id, 10); len(evs) != 0 {
			t.Fatalf("events leaked for %d", id)
		}
	}
}
