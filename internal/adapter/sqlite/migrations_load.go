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

var goHooks = map[string]func(*gorm.DB) error{}

func allMigrations() []migrate.Migration {
	migrationsOnce.Do(func() {
		loadedMigrations, loadMigrationsErr = migrate.LoadSQL(migrationFS, "migrations", goHooks)
	})

	if loadMigrationsErr != nil {
		panic("sqlite migrations: " + loadMigrationsErr.Error())
	}

	return loadedMigrations
}
