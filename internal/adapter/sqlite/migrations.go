package sqlite

import (
	"fmt"

	"gorm.io/gorm"
)

const schemaVersion = 3

type schemaMeta struct {
	ID      uint `gorm:"primaryKey"`
	Version int  `gorm:"not null;default:0"`
}

func (schemaMeta) TableName() string { return "schema_meta" }

func applyVersionedMigrations(db *gorm.DB) error {
	if err := db.AutoMigrate(&schemaMeta{}); err != nil {
		return fmt.Errorf("schema_meta: %w", err)
	}

	var meta schemaMeta
	if err := db.FirstOrCreate(&meta, schemaMeta{ID: 1}).Error; err != nil {
		return fmt.Errorf("schema_meta load: %w", err)
	}

	if meta.Version >= schemaVersion {
		return nil
	}

	if meta.Version < 1 {
		if err := db.Exec(`
			UPDATE repositories
			SET repo_path = name || '.git'
			WHERE deleted_at IS NULL
			  AND (repo_path IS NULL OR repo_path = '')
			  AND folder_id IS NULL
		`).Error; err != nil {
			return fmt.Errorf("migrate v1 root paths: %w", err)
		}
		meta.Version = 1
		if err := db.Save(&meta).Error; err != nil {
			return err
		}
	}

	if meta.Version < 2 {
		if err := db.Exec(`
			UPDATE repositories
			SET issues_enabled = 1
			WHERE deleted_at IS NULL
		`).Error; err != nil {
			return fmt.Errorf("migrate v2 issues_enabled: %w", err)
		}
		meta.Version = 2
		if err := db.Save(&meta).Error; err != nil {
			return err
		}
	}

	if meta.Version < 3 {
		if err := db.Exec(`
			UPDATE users
			SET role = 'user'
			WHERE deleted_at IS NULL
			  AND (role IS NULL OR role = '')
		`).Error; err != nil {
			return fmt.Errorf("migrate v3 user roles: %w", err)
		}
		meta.Version = 3
		if err := db.Save(&meta).Error; err != nil {
			return err
		}
	}

	return nil
}
