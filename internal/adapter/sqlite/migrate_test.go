package sqlite

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
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

	var version int
	if err := db.Raw(`SELECT MAX(version) FROM schema_migrations`).Scan(&version).Error; err != nil {
		t.Fatal(err)
	}

	if version != 0 {
		t.Fatalf("version %d", version)
	}

	store := NewRepositoryStore(db)
	repo, err := store.FindByOwnerAndRepoPath("user", 1, "missing.git")
	if err == nil || repo != nil {
		t.Fatal("expected missing repository")
	}

	again, err := NewDB(path)
	if err != nil {
		t.Fatal(err)
	}

	againSQL, err := again.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer againSQL.Close()

	baks, err := filepath.Glob(path + ".*.bak")
	if err != nil {
		t.Fatal(err)
	}

	if len(baks) != 1 {
		t.Fatalf("backups %d, want 1", len(baks))
	}
}

func TestMigrateLegacyRepositoryPreservesRow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "db.sqlite")

	raw, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`CREATE TABLE repositories (
		id INTEGER PRIMARY KEY,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME,
		name TEXT NOT NULL,
		owner_id INTEGER NOT NULL,
		description TEXT,
		path TEXT NOT NULL
	)`); err != nil {
		t.Fatal(err)
	}

	if _, err := raw.Exec(`INSERT INTO repositories (id, name, owner_id, description, path) VALUES (7, 'hello', 3, 'notes', '/data/hello.git')`); err != nil {
		t.Fatal(err)
	}

	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := NewDB(path)
	if err != nil {
		t.Fatal(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}

	defer sqlDB.Close()

	var got struct {
		Name     string
		OwnerID  uint
		Path     string
		RepoPath string
		FolderID uint
	}
	if err := db.Raw(`SELECT name, owner_id, path, repo_path, folder_id FROM repositories WHERE id = 7`).Scan(&got).Error; err != nil {
		t.Fatal(err)
	}

	if got.Name != "hello" || got.OwnerID != 3 || got.Path != "/data/hello.git" || got.RepoPath != "hello.git" || got.FolderID != 0 {
		t.Fatalf("row changed: %+v", got)
	}
}

func TestMigrateRejectsDuplicateRepoPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "db.sqlite")

	raw, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := raw.Exec(`CREATE TABLE repositories (
		id INTEGER PRIMARY KEY,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME,
		name TEXT NOT NULL,
		owner_id INTEGER NOT NULL,
		description TEXT,
		path TEXT NOT NULL,
		repo_path TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		t.Fatal(err)
	}

	if _, err := raw.Exec(`INSERT INTO repositories (id, name, owner_id, path, repo_path) VALUES (1, 'a', 1, '/a', 'same.git'), (2, 'b', 1, '/b', 'same.git')`); err != nil {
		t.Fatal(err)
	}

	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := NewDB(path); err == nil {
		t.Fatal("expected duplicate repo_path to stop migration")
	}

	raw, err = sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()

	var n int
	if err := raw.QueryRow(`SELECT COUNT(*) FROM repositories`).Scan(&n); err != nil {
		t.Fatal(err)
	}

	if n != 2 {
		t.Fatalf("rows %d", n)
	}

	var versions int
	if err := raw.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&versions); err != nil {
		t.Fatal(err)
	}

	if versions != 0 {
		t.Fatalf("recorded versions %d", versions)
	}
}
