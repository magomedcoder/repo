package migrate_test

import (
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"magomedcoder.ru/repo/pkg/migrate"
)

func TestValidateRejectsGap(t *testing.T) {
	t.Parallel()
	err := migrate.Validate([]migrate.Migration{
		{Version: 2, Name: "skip", Up: func(*gorm.DB) error { return nil }},
	})
	if err == nil || !strings.Contains(err.Error(), "contiguous") {
		t.Fatalf("err=%v", err)
	}
}

func TestFreshMarksAppliedWithoutRunningUp(t *testing.T) {
	t.Parallel()
	ran := false
	ms := []migrate.Migration{
		{
			Version: 1,
			Name:    "add_note",
			Up: func(db *gorm.DB) error {
				ran = true
				return db.Exec(`CREATE TABLE IF NOT EXISTS notes (id integer primary key, body text)`).Error
			},
		},
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

	if ran {
		t.Fatal("fresh bootstrap must not run Up")
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
		t.Fatalf("notes exists=%v err=%v", ok, err)
	}

	ok, err = migrate.TableExists(db, "seed")
	if err != nil || !ok {
		t.Fatalf("seed missing: %v", err)
	}
}

func TestApplyPendingOnExisting(t *testing.T) {
	t.Parallel()
	ms := []migrate.Migration{
		{
			Version: 1,
			Name:    "add_note",
			Up: func(db *gorm.DB) error {
				return db.Exec(`CREATE TABLE IF NOT EXISTS notes (id integer primary key, body text)`).Error
			},
		},
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

func TestDestructiveTriggersBackup(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := dir + "/db.sqlite"

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	if err := db.Exec(`CREATE TABLE seed (id integer primary key)`).Error; err != nil {
		t.Fatal(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()

	db2, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	if err := migrate.Run(migrate.Options{
		DB:     db2,
		DBPath: dbPath,
		Migrations: []migrate.Migration{
			{
				Version:     1,
				Name:        "boom",
				Destructive: true,
				Up: func(db *gorm.DB) error {
					return db.Exec(`CREATE TABLE notes (id integer primary key)`).Error
				},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	matches, err := filepath.Glob(filepath.Join(dir, "db.sqlite.bak.*"))
	if err != nil {
		t.Fatal(err)
	}

	if len(matches) != 1 {
		t.Fatalf("backups=%v", matches)
	}
}
