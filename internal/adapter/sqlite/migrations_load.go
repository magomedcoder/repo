package sqlite

import (
	"embed"
	"sync"

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
		loadedMigrations, loadMigrationsErr = migrate.LoadSQL(migrationFS, "migrations", nil)
	})

	if loadMigrationsErr != nil {
		panic("sqlite migrations: " + loadMigrationsErr.Error())
	}

	return loadedMigrations
}
