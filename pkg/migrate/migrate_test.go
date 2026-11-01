package migrate_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"magomedcoder.ru/repo/pkg/migrate"
)

func TestLoadSQLEmpty(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"migrations/.gitkeep": &fstest.MapFile{
			Data: []byte{},
		},
	}

	ms, err := migrate.LoadSQL(fsys, "migrations", nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(ms) != 0 {
		t.Fatalf("got %d migrations", len(ms))
	}
}

func TestFreshMarksAppliedWithoutRunningUp(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"migrations/0001_add_note.up.sql": &fstest.MapFile{
			Data: []byte(`CREATE TABLE IF NOT EXISTS notes (id integer primary key, body text);
`),
		},
	}
	ms, err := migrate.LoadSQL(fsys, "migrations", nil)
	if err != nil {
		t.Fatal(err)
	}

	db, err := gorm.Open(sqlite.Open("file:fresh?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	if err := migrate.Run(migrate.Options{
		DB:         db,
		Migrations: ms,
		Bootstrap: func(db *gorm.DB) error {
			return db.Exec(`CREATE TABLE seed (id integer primary key)`).Error
		},
	}); err != nil {
		t.Fatal(err)
	}

	applied, err := migrate.Applied(db)
	if err != nil {
		t.Fatal(err)
	}

	if len(applied) != 1 {
		t.Fatalf("applied=%d", len(applied))
	}

	ok, err := migrate.TableExists(db, "notes")
	if err != nil || ok {
		t.Fatalf("fresh bootstrap must not run Up SQL; notes exists=%v err=%v", ok, err)
	}

	ok, err = migrate.TableExists(db, "seed")
	if err != nil || !ok {
		t.Fatalf("seed missing: %v", err)
	}
}

func TestApplyPendingOnExisting(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"migrations/0001_add_note.up.sql": &fstest.MapFile{
			Data: []byte(`
CREATE TABLE IF NOT EXISTS notes (id integer primary key, body text);
`),
		},
	}
	ms, err := migrate.LoadSQL(fsys, "migrations", nil)
	if err != nil {
		t.Fatal(err)
	}

	db, err := gorm.Open(sqlite.Open("file:existing?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	if err := db.Exec(`CREATE TABLE seed (id integer primary key)`).Error; err != nil {
		t.Fatal(err)
	}

	if err := migrate.Run(migrate.Options{
		DB:         db,
		Migrations: ms,
	}); err != nil {
		t.Fatal(err)
	}

	ok, err := migrate.TableExists(db, "notes")
	if err != nil || !ok {
		t.Fatalf("notes missing: %v", err)
	}
}

func TestLoadSQLRejectsGap(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"migrations/0002_skip.up.sql": &fstest.MapFile{
			Data: []byte(`SELECT 1;`),
		},
	}

	_, err := migrate.LoadSQL(fsys, "migrations", nil)
	if err == nil || !strings.Contains(err.Error(), "contiguous") {
		t.Fatalf("err=%v", err)
	}
}
