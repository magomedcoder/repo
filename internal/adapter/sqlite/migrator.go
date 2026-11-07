package sqlite

import (
	"fmt"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/pkg/migrate"
)

func runMigrations(db *gorm.DB, _ string) error {
	empty, err := migrate.IsEmptySQLite(db)
	if err != nil {
		return err
	}

	if empty {
		if err := bootstrapSchema(db); err != nil {
			return err
		}
	}

	if err := ensureMissingTables(db); err != nil {
		return err
	}

	return VerifySchema(db)
}

func bootstrapSchema(db *gorm.DB) error {
	if err := db.AutoMigrate(bootstrapModels()...); err != nil {
		return fmt.Errorf("automigrate bootstrap: %w", err)
	}

	return ensureIndexes(db)
}

func ensureMissingTables(db *gorm.DB) error {
	m := db.Migrator()
	var missing []any
	for _, schemaModel := range bootstrapModels() {
		if !m.HasTable(schemaModel) {
			missing = append(missing, schemaModel)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	if err := db.AutoMigrate(missing...); err != nil {
		return fmt.Errorf("ensure missing tables: %w", err)
	}

	return ensureIndexes(db)
}

func ensureIndexes(db *gorm.DB) error {
	for _, stmt := range []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users (username) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email) WHERE deleted_at IS NULL`,
		`DROP INDEX IF EXISTS idx_repo_scope`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_repositories_name ON repositories (name) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_repositories_owner ON repositories (owner_id) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_repositories_repo_path ON repositories (repo_path) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_teams_owner_slug ON teams (owner_user_id, slug) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_team_members_team_user ON team_members (team_id, user_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_team_members_user ON team_members (user_id) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_team_repo_grants_team_repo ON team_repo_grants (team_id, repo_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_team_repo_grants_repo ON team_repo_grants (repo_id) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_repo_collaborators_repo_user ON repo_collaborators (repo_id, user_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_repo_collaborators_user ON repo_collaborators (user_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_activity_events_owner ON activity_events (owner_id)`,
		`CREATE INDEX IF NOT EXISTS idx_access_logs_created_at ON access_logs (created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_access_logs_action ON access_logs (action)`,
		`CREATE INDEX IF NOT EXISTS idx_repositories_mirror ON repositories (mirror_direction, mirror_status) WHERE mirror_direction != ''`,
		`CREATE INDEX IF NOT EXISTS idx_repositories_mirror_due ON repositories (mirror_enabled, mirror_next_sync_at) WHERE mirror_direction != ''`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("index: %w", err)
		}
	}

	return nil
}
