package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/magomedcoder/repo/internal/domain"
)

func TestRepositoryUpdateOptimisticLock(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatal(err)
	}

	users := NewUserRepo(db)
	owner := &domain.User{
		Username:     "owner",
		Email:        "owner@example.com",
		PasswordHash: "x",
		Role:         domain.RoleUser,
	}
	if err := users.Create(owner); err != nil {
		t.Fatal(err)
	}

	store := NewRepositoryRepo(db)
	repo := &domain.Repository{
		Name:          "r",
		OwnerID:       owner.ID,
		RepoPath:      "r.git",
		Path:          "/tmp/r.git",
		DefaultBranch: "main",
	}
	if err := store.Create(repo); err != nil {
		t.Fatal(err)
	}

	stale, err := store.FindByID(context.Background(), repo.ID)
	if err != nil {
		t.Fatal(err)
	}

	fresh, err := store.FindByID(context.Background(), repo.ID)
	if err != nil {
		t.Fatal(err)
	}

	fresh.IsPrivate = true
	if err := store.Update(fresh); err != nil {
		t.Fatalf("first update: %v", err)
	}

	stale.IsPrivate = false
	stale.Description = "rollback"
	if err := store.Update(stale); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("stale update must conflict, got %v", err)
	}

	got, err := store.FindByID(context.Background(), repo.ID)
	if err != nil {
		t.Fatal(err)
	}

	if !got.IsPrivate || got.Description != "" {
		t.Fatalf("stale write leaked: private=%v desc=%q", got.IsPrivate, got.Description)
	}

	if err := store.Delete(repo.ID); err != nil {
		t.Fatal(err)
	}

	got.Description = "ghost"
	if err := store.Update(got); err == nil {
		t.Fatal("update of a deleted repository must not upsert")
	}

	if _, err := store.FindByID(context.Background(), repo.ID); err == nil {
		t.Fatal("deleted repository must stay deleted")
	}
}

func TestUpdateMirrorStateDoesNotBumpUpdatedAt(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatal(err)
	}

	users := NewUserRepo(db)
	owner := &domain.User{
		Username:     "owner2",
		Email:        "owner2@example.com",
		PasswordHash: "x",
		Role:         domain.RoleUser,
	}
	if err := users.Create(owner); err != nil {
		t.Fatal(err)
	}

	store := NewRepositoryRepo(db)
	repo := &domain.Repository{
		Name:            "m",
		OwnerID:         owner.ID,
		RepoPath:        "m.git",
		Path:            "/tmp/m.git",
		DefaultBranch:   "main",
		MirrorDirection: domain.MirrorPull,
		MirrorURL:       "https://example.com/m.git",
		MirrorEnabled:   true,
	}
	if err := store.Create(repo); err != nil {
		t.Fatal(err)
	}

	before := repo.UpdatedAt
	now := before.Add(time.Hour)
	repo.MirrorLastSyncAt = &now
	repo.MirrorStatus = domain.MirrorStatusOK
	if err := store.UpdateMirrorState(context.Background(), repo); err != nil {
		t.Fatal(err)
	}

	got, err := store.FindByID(context.Background(), repo.ID)
	if err != nil {
		t.Fatal(err)
	}

	if !got.UpdatedAt.Equal(before) {
		t.Fatalf("mirror sync bumped updated_at: before=%v after=%v", before, got.UpdatedAt)
	}

	if got.MirrorStatus != domain.MirrorStatusOK {
		t.Fatalf("status=%q", got.MirrorStatus)
	}
}

func TestTouchLastActivityDoesNotBumpUpdatedAt(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatal(err)
	}

	users := NewUserRepo(db)
	owner := &domain.User{
		Username:     "owner3",
		Email:        "owner3@example.com",
		PasswordHash: "x",
		Role:         domain.RoleUser,
	}
	if err := users.Create(owner); err != nil {
		t.Fatal(err)
	}

	store := NewRepositoryRepo(db)
	repo := &domain.Repository{
		Name:          "p",
		OwnerID:       owner.ID,
		RepoPath:      "p.git",
		Path:          "/tmp/p.git",
		DefaultBranch: "main",
	}
	if err := store.Create(repo); err != nil {
		t.Fatal(err)
	}

	before := repo.UpdatedAt
	at := before.Add(2 * time.Hour)
	if err := store.TouchLastActivity(repo.ID, at); err != nil {
		t.Fatal(err)
	}

	got, err := store.FindByID(context.Background(), repo.ID)
	if err != nil {
		t.Fatal(err)
	}

	if !got.UpdatedAt.Equal(before) {
		t.Fatalf("touch bumped updated_at: before=%v after=%v", before, got.UpdatedAt)
	}

	if got.LastActivityAt == nil || !got.LastActivityAt.Equal(at) {
		t.Fatalf("last_activity_at=%v want %v", got.LastActivityAt, at)
	}
}
