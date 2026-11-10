package migrate

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

func ensureMigrationsTable(db *gorm.DB) error {
	const stmt = `CREATE TABLE IF NOT EXISTS schema_migrations (name TEXT PRIMARY KEY NOT NULL, applied_at TEXT NOT NULL)`
	if err := db.Exec(stmt).Error; err != nil {
		return fmt.Errorf("schema_migrations: %w", err)
	}

	return nil
}

func appliedMigrationNames(db *gorm.DB) (map[string]struct{}, error) {
	var names []string
	if err := db.Raw(`SELECT name FROM schema_migrations`).Scan(&names).Error; err != nil {
		return nil, fmt.Errorf("list schema_migrations: %w", err)
	}

	out := make(map[string]struct{}, len(names))
	for _, name := range names {
		out[name] = struct{}{}
	}

	return out, nil
}

func recordMigration(db *gorm.DB, name string) error {
	if err := db.Exec(`INSERT INTO schema_migrations (name, applied_at) VALUES (?, ?)`, name, time.Now().UTC().Format(time.RFC3339Nano)).Error; err != nil {
		return fmt.Errorf("record migration %s: %w", name, err)
	}

	return nil
}
