package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/magomedcoder/repo/internal/domain"
)

func TestMaintenancePurges(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	db, err := NewDB(dir)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	now := time.Now()

	users := NewUserRepo(db)
	u := &domain.User{
		Username:     "utest2",
		Email:        "utest2@example.com",
		PasswordHash: "x",
	}
	if err := users.Create(u); err != nil {
		t.Fatal(err)
	}

	sessions := NewSessionRepo(db)
	for i, exp := range []time.Time{now.Add(-time.Hour), now.Add(time.Hour)} {
		if err := sessions.Create(&domain.Session{
			Token:     string(rune('a' + i)),
			UserID:    u.ID,
			ExpiresAt: exp,
		}); err != nil {
			t.Fatal(err)
		}
	}

	if n, err := sessions.DeleteExpired(ctx, now); err != nil || n != 1 {
		t.Fatalf("sessions: n=%d err=%v", n, err)
	}

	tokens := NewAccessTokenRepo(db)
	past, future := now.Add(-time.Hour), now.Add(time.Hour)
	for i, exp := range []*time.Time{&past, &future, nil} {
		if err := tokens.Create(&domain.AccessToken{
			UserID:    u.ID,
			Name:      "t",
			TokenHash: string(rune('a' + i)),
			Prefix:    "p",
			ExpiresAt: exp,
		}); err != nil {
			t.Fatal(err)
		}
	}

	if n, err := tokens.DeleteExpired(ctx, now); err != nil || n != 1 {
		t.Fatalf("tokens: n=%d err=%v", n, err)
	}

	acts := NewActivityRepo(db)
	ev := &domain.ActivityEvent{
		ActorID: u.ID,
		OwnerID: u.ID,
		Type:    domain.ActivityPush,
		Summary: "x",
	}
	if err := acts.Create(ev); err != nil {
		t.Fatal(err)
	}

	if n, err := acts.DeleteOlderThan(ctx, now.Add(-time.Hour)); err != nil || n != 0 {
		t.Fatalf("activity keep: n=%d err=%v", n, err)
	}

	if n, err := acts.DeleteOlderThan(ctx, now.Add(time.Hour)); err != nil || n != 1 {
		t.Fatalf("activity purge: n=%d err=%v", n, err)
	}

	if err := CheckpointWAL(db); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
}

func TestPurgeSoftDeleted(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	db, err := NewDB(dir)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	users := NewUserRepo(db)
	u := &domain.User{
		Username:     "carol",
		Email:        "carol@example.com",
		PasswordHash: "x",
	}
	if err := users.Create(u); err != nil {
		t.Fatal(err)
	}

	repos := NewRepositoryRepo(db)
	mk := func(name string) *domain.Repository {
		r := &domain.Repository{
			Name:          name,
			OwnerID:       u.ID,
			DefaultBranch: "main",
			Path:          filepath.Join(dir, name+".git"),
			RepoPath:      name + ".git",
		}
		if err := repos.Create(r); err != nil {
			t.Fatal(err)
		}

		return r
	}

	gone, kept, live := mk("gone"), mk("kept"), mk("live")
	acts := NewActivityRepo(db)
	for _, r := range []*domain.Repository{gone, live} {
		id := r.ID
		if err := acts.Create(&domain.ActivityEvent{
			ActorID: u.ID,
			OwnerID: u.ID,
			RepoID:  &id,
			Type:    domain.ActivityPush,
			Summary: "x",
		}); err != nil {
			t.Fatal(err)
		}
	}

	events := NewMirrorSyncRepo(db)
	if err := events.Create(ctx, &domain.MirrorSyncEvent{
		RepoID:    gone.ID,
		Status:    domain.MirrorEventOK,
		StartedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	if err := repos.Delete(gone.ID); err != nil {
		t.Fatal(err)
	}

	if err := repos.Delete(kept.ID); err != nil {
		t.Fatal(err)
	}

	old := time.Now().Add(-8 * 24 * time.Hour)
	if err := db.Exec(`UPDATE repositories SET deleted_at = ? WHERE id = ?`, old, gone.ID).Error; err != nil {
		t.Fatal(err)
	}

	paths, ids, err := repos.PurgeSoftDeleted(ctx, time.Now().Add(-7*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	if len(ids) != 1 || ids[0] != gone.ID {
		t.Fatalf("purged ids=%v want [%d]", ids, gone.ID)
	}
	if len(paths) != 1 || paths[0] == "" {
		t.Fatalf("purged paths=%v", paths)
	}

	var n int64
	db.Raw(`SELECT COUNT(*) FROM repositories`).Scan(&n)
	if n != 2 {
		t.Fatalf("repositories rows=%d want 2 (kept soft-deleted + live)", n)
	}

	db.Raw(`SELECT COUNT(*) FROM activity_events WHERE repo_id = ?`, gone.ID).Scan(&n)
	if n != 0 {
		t.Fatalf("activity for purged repo remains: %d", n)
	}

	db.Raw(`SELECT COUNT(*) FROM activity_events WHERE repo_id = ?`, live.ID).Scan(&n)
	if n != 1 {
		t.Fatalf("activity for live repo lost: %d", n)
	}

	db.Raw(`SELECT COUNT(*) FROM mirror_sync_events WHERE repo_id = ?`, gone.ID).Scan(&n)
	if n != 0 {
		t.Fatalf("mirror events remain: %d", n)
	}
}
