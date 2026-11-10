package migrate_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/magomedcoder/repo/internal/adapter/sqlite/migrate"
	"github.com/magomedcoder/repo/pkg/sqliteutil"
	"gorm.io/gorm"
)

func openDB(t *testing.T, dir string) *gorm.DB {
	t.Helper()
	db, err := sqliteutil.Open(filepath.Join(dir, "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func migratedDB(t *testing.T) (string, *gorm.DB) {
	t.Helper()
	dir := t.TempDir()
	db := openDB(t, dir)
	if err := migrate.Run(db, nil); err != nil {
		t.Fatal(err)
	}

	return dir, db
}

func TestSchemaMatchesModelsFresh(t *testing.T) {
	t.Parallel()
	_, db := migratedDB(t)
	if err := migrate.VerifySchema(db); err != nil {
		t.Fatal(err)
	}
}

func TestSchemaDetectsMissingColumn(t *testing.T) {
	t.Parallel()
	_, db := migratedDB(t)

	if err := db.Exec(`ALTER TABLE repositories DROP COLUMN mirror_remote_head`).Error; err != nil {
		t.Fatal(err)
	}

	err := migrate.VerifySchema(db)
	if err == nil || !strings.Contains(err.Error(), "mirror_remote_head") {
		t.Fatalf("want missing mirror_remote_head, got %v", err)
	}
}

func TestDropShowOnHomeColumn(t *testing.T) {
	t.Parallel()
	dir, db := migratedDB(t)

	if err := db.Exec(`ALTER TABLE repositories ADD COLUMN show_on_home INTEGER NOT NULL DEFAULT 0`).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Exec(`DELETE FROM schema_migrations WHERE name = ?`, "drop_repositories_show_on_home").Error; err != nil {
		t.Fatal(err)
	}

	ok, err := migrate.TableHasColumn(db, "repositories", "show_on_home")
	if err != nil || !ok {
		t.Fatalf("expected show_on_home before upgrade: ok=%v err=%v", ok, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()

	db2 := openDB(t, dir)
	if err := migrate.Run(db2, nil); err != nil {
		t.Fatal(err)
	}

	ok, err = migrate.TableHasColumn(db2, "repositories", "show_on_home")
	if err != nil {
		t.Fatal(err)
	}

	if ok {
		t.Fatal("show_on_home should be dropped")
	}

	var n int64
	if err := db2.Raw(`SELECT COUNT(1) FROM schema_migrations WHERE name = ?`, "drop_repositories_show_on_home").Scan(&n).Error; err != nil {
		t.Fatal(err)
	}

	if n != 1 {
		t.Fatalf("migration not recorded: %d", n)
	}
}

func TestSkipAlreadyAppliedMigration(t *testing.T) {
	t.Parallel()
	dir, db := migratedDB(t)

	if err := db.Exec(`ALTER TABLE repositories ADD COLUMN show_on_home INTEGER NOT NULL DEFAULT 0`).Error; err != nil {
		t.Fatal(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()

	db2 := openDB(t, dir)
	if err := migrate.Run(db2, nil); err != nil {
		t.Fatal(err)
	}

	ok, err := migrate.TableHasColumn(db2, "repositories", "show_on_home")
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("already-applied migration must not run again")
	}
}

func TestMigrationsJournalOnFreshDB(t *testing.T) {
	t.Parallel()
	_, db := migratedDB(t)

	var n int64
	if err := db.Raw(`SELECT COUNT(1) FROM schema_migrations`).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}

	if n < 1 {
		t.Fatalf("expected journal rows, got %d", n)
	}
}
