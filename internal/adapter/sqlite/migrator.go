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
		&folderMemberModel{},
		&teamModel{},
		&teamMemberModel{},
		&teamRepoGrantModel{},
		&repoCollaboratorModel{},
		&issueModel{},
		&issueCommentModel{},
		&labelModel{},
		&issueLabelModel{},
		&pullRequestModel{},
		&pullCommentModel{},
		&activityModel{},
		&accessLogModel{},
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
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_folders_path ON folders (path) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_folders_slug ON folders (slug) WHERE deleted_at IS NULL AND parent_id IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_folders_owner_path ON folders (owner_kind, owner_id, path)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_repo_scope ON repositories (owner_kind, owner_id, ifnull(folder_id, 0), name) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_repositories_repo_path ON repositories (repo_path) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_folder_members_folder_user ON folder_members (folder_id, user_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_folder_members_user ON folder_members (user_id) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_teams_folder_slug ON teams (folder_id, slug) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_team_members_team_user ON team_members (team_id, user_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_team_members_user ON team_members (user_id) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_team_repo_grants_team_repo ON team_repo_grants (team_id, repo_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_team_repo_grants_repo ON team_repo_grants (repo_id) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_repo_collaborators_repo_user ON repo_collaborators (repo_id, user_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_repo_collaborators_user ON repo_collaborators (user_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_activity_events_owner ON activity_events (owner_kind, owner_id)`,
		`CREATE INDEX IF NOT EXISTS idx_access_logs_created_at ON access_logs (created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_access_logs_action ON access_logs (action)`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("index: %w", err)
		}
	}

	return nil
}
