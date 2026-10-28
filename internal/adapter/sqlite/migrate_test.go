package sqlite

import (
	"path/filepath"
	"testing"

	"repo.magomedcoder.ru/repo/internal/domain"
)

func TestMigrateFreshDatabase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "db.sqlite")

	db, err := NewDB(path)
	if err != nil {
		t.Fatal(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	store := NewRepositoryStore(db)
	repo, err := store.FindByRepoPath("missing.git")
	if err == nil || repo != nil {
		t.Fatal("expected missing repository")
	}

	if _, err := NewDB(path); err != nil {
		t.Fatal(err)
	}
}

func TestSoftDeleteAllowsNameReuse(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}

	users := NewUserStore(db)
	u := &domain.User{
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "x",
	}
	if err := users.Create(u); err != nil {
		t.Fatal(err)
	}

	first := &domain.Repository{
		Name:          "demo",
		OwnerKind:     domain.OwnerKindUser,
		OwnerID:       u.ID,
		DefaultBranch: "main",
		Path:          filepath.Join(dir, "demo.git"),
		RepoPath:      "demo.git",
	}
	repos := NewRepositoryStore(db)
	if err := repos.Create(first); err != nil {
		t.Fatal(err)
	}
	if err := repos.Delete(first.ID); err != nil {
		t.Fatal(err)
	}

	second := &domain.Repository{
		Name:          "demo",
		OwnerKind:     domain.OwnerKindUser,
		OwnerID:       u.ID,
		DefaultBranch: "main",
		Path:          filepath.Join(dir, "demo2.git"),
		RepoPath:      "demo.git",
	}
	if err := repos.Create(second); err != nil {
		t.Fatalf("reuse after soft-delete: %v", err)
	}
}
