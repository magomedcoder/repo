package sqlite

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/adapter/sqlite/migrations"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/migrate"
)

func TestMigrateFreshDatabase(t *testing.T) {
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

	var count int64
	if err := db.Model(&migrate.Record{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if int(count) != len(migrations.All()) {
		t.Fatalf("journal rows=%d want %d (%v)", count, len(migrations.All()), migrate.Names(migrations.All()))
	}

	ok, err := migrate.ColumnExists(db, "users", "role")
	if err != nil || !ok {
		t.Fatalf("users.role missing: %v", err)
	}

	ok, err = migrate.ColumnExists(db, "repositories", "issues_enabled")
	if err != nil || !ok {
		t.Fatalf("repositories.issues_enabled missing: %v", err)
	}

	store := NewRepositoryStore(db)
	repo, err := store.FindByRepoPath(context.Background(), "missing.git")
	if err == nil || repo != nil {
		t.Fatal("expected missing repository")
	}

	if _, err := NewDB(dir); err != nil {
		t.Fatal(err)
	}
}

func TestMigrateIdempotentOnExistingDB(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "db.sqlite")

	db, err := openRawDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	if err := bootstrapSchema(db); err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(&migrate.Record{}); err != nil {
		t.Fatal(err)
	}

	users := NewUserStore(db)
	u := &domain.User{
		Username:     "bob",
		Email:        "bob@example.com",
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

	got, err := NewUserStore(db2).FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}

	if got.Username != "bob" || got.Role != domain.RoleUser {
		t.Fatalf("user not preserved: %+v", got)
	}
}

func TestMigrateV1AddsMirrorsAndAccessTables(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "db.sqlite")
	db, err := openRawDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	for _, stmt := range []string{
		`CREATE TABLE users (id integer PRIMARY KEY AUTOINCREMENT, username text, email text, password_hash text, role text, deleted_at datetime)`,
		`CREATE TABLE folders (
			id integer PRIMARY KEY AUTOINCREMENT,
			created_at datetime, updated_at datetime, deleted_at datetime,
			owner_kind text NOT NULL DEFAULT "user", owner_id integer NOT NULL,
			name text NOT NULL, slug text NOT NULL, path text NOT NULL
		)`,
		`CREATE TABLE repositories (
			id integer PRIMARY KEY AUTOINCREMENT,
			created_at datetime, updated_at datetime, deleted_at datetime,
			name text NOT NULL, owner_kind text NOT NULL DEFAULT "user", owner_id integer NOT NULL,
			folder_id integer, description text, is_private numeric, show_on_home numeric,
			protect_default_branch numeric, issues_enabled numeric,
			default_branch text, path text NOT NULL, repo_path text,
			issue_seq integer NOT NULL DEFAULT 0, pull_seq integer NOT NULL DEFAULT 0,
			last_activity_at datetime
		)`,
		`CREATE TABLE issues (id integer PRIMARY KEY AUTOINCREMENT, repository_id integer, number integer)`,
		`CREATE TABLE pull_requests (id integer PRIMARY KEY AUTOINCREMENT, repository_id integer, number integer)`,
		`CREATE TABLE access_tokens (id integer PRIMARY KEY AUTOINCREMENT, user_id integer)`,
		`INSERT INTO folders (owner_id, name, slug, path) VALUES (1, 'work', 'work', 'work')`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatal(err)
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()

	db2, err := NewDB(dir)
	if err != nil {
		t.Fatalf("migrate v1: %v", err)
	}
	sqlDB2, err := db2.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB2.Close()

	ok, err := migrate.ColumnExists(db2, "repositories", "mirror_direction")
	if err != nil || !ok {
		t.Fatalf("mirror_direction after v1: ok=%v err=%v", ok, err)
	}

	ok, err = migrate.TableExists(db2, "folder_members")
	if err != nil || !ok {
		t.Fatalf("folder_members after v1: ok=%v err=%v", ok, err)
	}

	var versions int64
	if err := db2.Model(&migrate.Record{}).Count(&versions).Error; err != nil || versions != 1 {
		t.Fatalf("journal versions=%d err=%v", versions, err)
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

func openRawDB(dbPath string) (*gorm.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		return nil, err
	}

	dsn := dbPath
	if !strings.Contains(dsn, "_pragma") {
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		dsn += sep + "_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(FULL)"
	}
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	return db, nil
}
