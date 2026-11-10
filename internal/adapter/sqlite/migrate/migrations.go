package migrate

import (
	"fmt"

	"gorm.io/gorm"
)

type schemaMigration struct {
	Name string
	Up   func(*gorm.DB) error
}

var schemaMigrations = []schemaMigration{
	{Name: "drop_repositories_show_on_home", Up: migDropRepositoriesShowOnHome},
}

func applySchemaUpgrades(db *gorm.DB) (int, error) {
	if err := ensureMigrationsTable(db); err != nil {
		return 0, err
	}

	done, err := appliedMigrationNames(db)
	if err != nil {
		return 0, err
	}

	applied := 0
	for _, m := range schemaMigrations {
		if _, ok := done[m.Name]; ok {
			continue
		}

		if err := m.Up(db); err != nil {
			return applied, fmt.Errorf("migration %s: %w", m.Name, err)
		}

		if err := recordMigration(db, m.Name); err != nil {
			return applied, err
		}

		applied++
	}

	return applied, nil
}

func migDropRepositoriesShowOnHome(db *gorm.DB) error {
	return dropColumnIfExists(db, "repositories", "show_on_home")
}
