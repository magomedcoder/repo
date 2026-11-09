package sqlite

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/magomedcoder/repo/internal/adapter/sqlite/model"
	"github.com/magomedcoder/repo/internal/domain"
)

func TestBootstrapFreshDatabase(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	db, err := NewDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	if !db.Migrator().HasColumn(&model.User{}, "role") {
		t.Fatal("users.role missing")
	}

	if !db.Migrator().HasColumn(&model.Repository{}, "issues_enabled") {
		t.Fatal("repositories.issues_enabled missing")
	}

	store := NewRepositoryRepo(db)
	repo, err := store.FindByRepoPath(context.Background(), "missing.git")
	if err == nil || repo != nil {
		t.Fatal("expected missing repository")
	}

	if _, err := NewDB(dir); err != nil {
		t.Fatal(err)
	}
}

func TestBootstrapIdempotentOnExistingDB(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	db, err := NewDB(dir)
	if err != nil {
		t.Fatal(err)
	}

	users := NewUserRepo(db)
	u := &domain.User{
		Username:     "utest2",
		Email:        "utest2@example.com",
		PasswordHash: "x",
		Role:         domain.RoleUser,
	}
	if err := users.Create(u); err != nil {
		t.Fatal(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()

	db2, err := NewDB(dir)
	if err != nil {
		t.Fatal(err)
	}

	sqlDB2, err := db2.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB2.Close()

	got, err := NewUserRepo(db2).FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}

	if got.Username != "utest2" || got.Role != domain.RoleUser {
		t.Fatalf("user not preserved: %+v", got)
	}
}

func TestGlobalNamesAndSoftDelete(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	db, err := NewDB(dir)
	if err != nil {
		t.Fatal(err)
	}

	users := NewUserRepo(db)
	u1 := &domain.User{
		Username:     "utest",
		Email:        "utest@example.com",
		PasswordHash: "x",
	}
	u2 := &domain.User{
		Username:     "utest2",
		Email:        "utest2@example.com",
		PasswordHash: "x",
	}
	for _, u := range []*domain.User{u1, u2} {
		if err := users.Create(u); err != nil {
			t.Fatal(err)
		}
	}

	repos := NewRepositoryRepo(db)
	first := &domain.Repository{
		Name:          "demo",
		OwnerID:       u1.ID,
		DefaultBranch: "main",
		Path:          filepath.Join(dir, "demo.git"),
		RepoPath:      "demo.git",
	}
	if err := repos.Create(first); err != nil {
		t.Fatal(err)
	}

	second := &domain.Repository{
		Name:          "Demo",
		OwnerID:       u2.ID,
		DefaultBranch: "main",
		Path:          filepath.Join(dir, "Demo.git"),
		RepoPath:      "Demo.git",
	}
	if err := repos.Create(second); err == nil {
		t.Fatal("same name for another owner must conflict (global, case-insensitive)")
	}

	other := &domain.Repository{
		Name:          "other",
		OwnerID:       u2.ID,
		DefaultBranch: "main",
		Path:          filepath.Join(dir, "other.git"),
		RepoPath:      "other.git",
	}
	if err := repos.Create(other); err != nil {
		t.Fatal(err)
	}

	if ok, err := repos.ExistsByName("demo"); err != nil || !ok {
		t.Fatalf("exists=%v err=%v", ok, err)
	}

	if ok, err := repos.ExistsByName("missing"); err != nil || ok {
		t.Fatalf("unexpected exists=%v err=%v", ok, err)
	}

	if err := repos.Delete(first.ID); err != nil {
		t.Fatal(err)
	}

	if ok, err := repos.ExistsByName("demo"); err != nil || ok {
		t.Fatalf("soft-deleted row must not collide at store level (holds do), exists=%v err=%v", ok, err)
	}
}

func TestRepoNameHoldStore(t *testing.T) {
	t.Parallel()
	db, err := NewDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	holds := NewRepoNameHoldRepo(db)
	ctx := context.Background()
	now := time.Now().UTC()

	if err := holds.Hold(ctx, "Old-Name", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	if err := holds.Hold(ctx, "old-name", now.Add(2*time.Hour)); err != nil {
		t.Fatalf("re-hold must upsert: %v", err)
	}

	if ok, err := holds.IsHeld(ctx, "old-name", now); err != nil || !ok {
		t.Fatalf("held=%v err=%v", ok, err)
	}

	if ok, _ := holds.IsHeld(ctx, "old-name", now.Add(3*time.Hour)); ok {
		t.Fatal("expired hold must not block")
	}

	n, err := holds.DeleteExpired(ctx, now.Add(3*time.Hour))
	if err != nil || n != 1 {
		t.Fatalf("deleted=%d err=%v", n, err)
	}
}

func TestSchemaMatchesModelsFresh(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	db, err := NewDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	if err := VerifySchema(db); err != nil {
		t.Fatal(err)
	}
}

func TestSchemaDetectsMissingColumn(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	db, err := NewDB(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.Exec(`ALTER TABLE repositories DROP COLUMN mirror_remote_head`).Error; err != nil {
		t.Fatal(err)
	}

	err = VerifySchema(db)
	if err == nil || !strings.Contains(err.Error(), "mirror_remote_head") {
		t.Fatalf("want missing mirror_remote_head, got %v", err)
	}

	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
}
