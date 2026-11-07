package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/migrate"
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

	ok, err := migrate.ColumnExists(db, "users", "role")
	if err != nil || !ok {
		t.Fatalf("users.role missing: %v", err)
	}

	ok, err = migrate.ColumnExists(db, "repositories", "issues_enabled")
	if err != nil || !ok {
		t.Fatalf("repositories.issues_enabled missing: %v", err)
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

func TestBackupSQLite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := filepath.Join(dir, "db.sqlite")
	if err := os.WriteFile(src, []byte("sqlite-demo"), 0o600); err != nil {
		t.Fatal(err)
	}

	dst, err := migrate.BackupSQLite(src)
	if err != nil {
		t.Fatal(err)
	}

	if dst == "" {
		t.Fatal("expected backup path")
	}

	body, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "sqlite-demo" {
		t.Fatalf("backup body=%q", body)
	}
}

func TestSoftDeleteAllowsNameReuse(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	db, err := NewDB(dir)
	if err != nil {
		t.Fatal(err)
	}

	users := NewUserRepo(db)
	u := &domain.User{
		Username:     "utest",
		Email:        "utest@example.com",
		PasswordHash: "x",
	}
	if err := users.Create(u); err != nil {
		t.Fatal(err)
	}

	first := &domain.Repository{
		Name:          "demo",
		OwnerID:       u.ID,
		DefaultBranch: "main",
		Path:          filepath.Join(dir, "demo.git"),
		RepoPath:      "demo.git",
	}
	repos := NewRepositoryRepo(db)
	if err := repos.Create(first); err != nil {
		t.Fatal(err)
	}
	if err := repos.Delete(first.ID); err != nil {
		t.Fatal(err)
	}

	exists, err := repos.ExistsByName("demo")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("soft-deleted name must still be reserved until purge")
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

	if err := db.Exec(`ALTER TABLE repositories DROP COLUMN mirror_partial`).Error; err != nil {
		t.Fatal(err)
	}

	err = VerifySchema(db)
	if err == nil || !strings.Contains(err.Error(), "mirror_partial") {
		t.Fatalf("want missing mirror_partial, got %v", err)
	}

	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
}
