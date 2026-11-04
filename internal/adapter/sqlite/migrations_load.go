package sqlite

import (
	"embed"
	"sync"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/pkg/migrate"
)

//go:embed all:migrations
var migrationFS embed.FS

var (
	migrationsOnce    sync.Once
	loadedMigrations  []migrate.Migration
	loadMigrationsErr error
)

func allMigrations() []migrate.Migration {
	migrationsOnce.Do(func() {
		loadedMigrations, loadMigrationsErr = migrate.LoadSQL(migrationFS, "migrations", map[string]func(*gorm.DB) error{
			"add_token_expiry_and_seqs": addTokenExpiryAndSeqs,
		})
	})

	if loadMigrationsErr != nil {
		panic("sqlite migrations: " + loadMigrationsErr.Error())
	}

	return loadedMigrations
}

func addTokenExpiryAndSeqs(db *gorm.DB) error {
	if err := migrate.AddColumnIfMissing(db, "access_tokens", "expires_at", `ALTER TABLE access_tokens ADD COLUMN expires_at DATETIME`); err != nil {
		return err
	}

	if err := migrate.AddColumnIfMissing(db, "repositories", "issue_seq", `ALTER TABLE repositories ADD COLUMN issue_seq INTEGER NOT NULL DEFAULT 0`); err != nil {
		return err
	}

	if err := migrate.AddColumnIfMissing(db, "repositories", "pull_seq", `ALTER TABLE repositories ADD COLUMN pull_seq INTEGER NOT NULL DEFAULT 0`); err != nil {
		return err
	}

	return nil
}
