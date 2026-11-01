package sqlite

import (
	"fmt"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/pkg/migrate"
)

func runMigrations(db *gorm.DB, dbPath string) error {
	return migrate.Run(migrate.Options{
		DB:         db,
		DBPath:     dbPath,
		Migrations: allMigrations(),
		Bootstrap:  bootstrapSchema,
	})
}

func bootstrapSchema(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&userModel{},
		&sessionModel{},
		&accessTokenModel{},
		&folderModel{},
		&repositoryModel{},
		&issueModel{},
		&issueCommentModel{},
		&labelModel{},
		&issueLabelModel{},
		&pullRequestModel{},
		&pullCommentModel{},
		&activityModel{},
	); err != nil {
		return fmt.Errorf("automigrate bootstrap: %w", err)
	}

	return ensureIndexes(db)
}

func ensureIndexes(db *gorm.DB) error {
	for _, stmt := range []string{
		`DROP INDEX IF EXISTS idx_folder_scope`,
		`DROP INDEX IF EXISTS idx_repo_scope`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users (username) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_folder_scope ON folders (owner_kind, owner_id, ifnull(parent_id, 0), slug) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_folders_owner_path ON folders (owner_kind, owner_id, path)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_repo_scope ON repositories (owner_kind, owner_id, ifnull(folder_id, 0), name) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_repositories_repo_path ON repositories (repo_path) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_activity_events_owner ON activity_events (owner_kind, owner_id)`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("index: %w", err)
		}
	}

	return nil
}
