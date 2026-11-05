package migrations

import (
	"gorm.io/gorm"
	"magomedcoder.ru/repo/pkg/migrate"
)

func v1() migrate.Migration {
	return migrate.Migration{
		Version:     1,
		Name:        "migration",
		Destructive: true,
		Up:          upV1,
	}
}

func upV1(db *gorm.DB) error {
	if err := addTokenExpiryAndSeqs(db); err != nil {
		return err
	}
	if err := addSessionClientColumns(db); err != nil {
		return err
	}

	if err := dropFolderParentID(db); err != nil {
		return err
	}

	if err := flattenFolderPaths(db); err != nil {
		return err
	}

	if err := createAccessModelTables(db); err != nil {
		return err
	}

	if err := seedFolderOwners(db); err != nil {
		return err
	}

	if err := ensureFolderIndexes(db); err != nil {
		return err
	}

	if err := backfillIssuePullSeqs(db); err != nil {
		return err
	}

	return addRepoMirrorColumns(db)
}

func addTokenExpiryAndSeqs(db *gorm.DB) error {
	if err := migrate.AddColumnIfMissing(db, "access_tokens", "expires_at", `ALTER TABLE access_tokens ADD COLUMN expires_at DATETIME`); err != nil {
		return err
	}
	if err := migrate.AddColumnIfMissing(db, "repositories", "issue_seq", `ALTER TABLE repositories ADD COLUMN issue_seq INTEGER NOT NULL DEFAULT 0`); err != nil {
		return err
	}
	return migrate.AddColumnIfMissing(db, "repositories", "pull_seq", `ALTER TABLE repositories ADD COLUMN pull_seq INTEGER NOT NULL DEFAULT 0`)
}

func addSessionClientColumns(db *gorm.DB) error {
	if err := migrate.AddColumnIfMissing(db, "sessions", "ip", `ALTER TABLE sessions ADD COLUMN ip TEXT NOT NULL DEFAULT ''`); err != nil {
		return err
	}
	return migrate.AddColumnIfMissing(db, "sessions", "user_agent", `ALTER TABLE sessions ADD COLUMN user_agent TEXT NOT NULL DEFAULT ''`)
}

func dropFolderParentID(db *gorm.DB) error {
	ok, err := migrate.ColumnExists(db, "folders", "parent_id")
	if err != nil {
		return err
	}

	if !ok {
		return nil
	}

	for _, stmt := range []string{
		`PRAGMA foreign_keys = OFF`,
		`DROP INDEX IF EXISTS idx_folder_scope`,
		`DROP INDEX IF EXISTS idx_folders_slug`,
		`DROP INDEX IF EXISTS idx_folders_path`,
		`DROP INDEX IF EXISTS idx_folders_owner_path`,
		`CREATE TABLE folders__new (
			id integer PRIMARY KEY AUTOINCREMENT,
			created_at datetime,
			updated_at datetime,
			deleted_at datetime,
			owner_kind text NOT NULL DEFAULT "user",
			owner_id integer NOT NULL,
			name text NOT NULL,
			slug text NOT NULL,
			path text NOT NULL
		)`,
		`INSERT INTO folders__new (id, created_at, updated_at, deleted_at, owner_kind, owner_id, name, slug, path)
		 SELECT id, created_at, updated_at, deleted_at, owner_kind, owner_id, name, slug, path FROM folders`,
		`DROP TABLE folders`,
		`ALTER TABLE folders__new RENAME TO folders`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_folders_path ON folders (path) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_folders_owner_path ON folders (owner_kind, owner_id, path)`,
		`PRAGMA foreign_keys = ON`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func flattenFolderPaths(db *gorm.DB) error {
	if err := db.Exec(`UPDATE folders SET path = slug WHERE instr(path, '/') > 0`).Error; err != nil {
		return err
	}

	return db.Exec(`
		UPDATE repositories
		SET repo_path = CASE
			WHEN folder_id IS NULL OR folder_id = 0 THEN name || '.git'
			ELSE (
				SELECT f.path || '/' || repositories.name || '.git'
				FROM folders f
				WHERE f.id = repositories.folder_id
			)
		END
		WHERE deleted_at IS NULL
	`).Error
}

func createAccessModelTables(db *gorm.DB) error {
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS folder_members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			folder_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			role TEXT NOT NULL,
			caps INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_folder_members_folder_user ON folder_members (folder_id, user_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_folder_members_user ON folder_members (user_id) WHERE deleted_at IS NULL`,
		`CREATE TABLE IF NOT EXISTS teams (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			folder_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			caps INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_teams_folder_slug ON teams (folder_id, slug) WHERE deleted_at IS NULL`,
		`CREATE TABLE IF NOT EXISTS team_members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME,
			deleted_at DATETIME,
			team_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_team_members_team_user ON team_members (team_id, user_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_team_members_user ON team_members (user_id) WHERE deleted_at IS NULL`,
		`CREATE TABLE IF NOT EXISTS team_repo_grants (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			team_id INTEGER NOT NULL,
			repo_id INTEGER NOT NULL,
			role TEXT NOT NULL
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_team_repo_grants_team_repo ON team_repo_grants (team_id, repo_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_team_repo_grants_repo ON team_repo_grants (repo_id) WHERE deleted_at IS NULL`,
		`CREATE TABLE IF NOT EXISTS repo_collaborators (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			repo_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			role TEXT NOT NULL
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_repo_collaborators_repo_user ON repo_collaborators (repo_id, user_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_repo_collaborators_user ON repo_collaborators (user_id) WHERE deleted_at IS NULL`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedFolderOwners(db *gorm.DB) error {
	return db.Exec(`
		INSERT INTO folder_members (created_at, updated_at, folder_id, user_id, role, caps)
		SELECT CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, f.id, f.owner_id, 'owner', 127
		FROM folders f
		WHERE f.deleted_at IS NULL
		  AND f.owner_id IS NOT NULL
		  AND f.owner_id > 0
		  AND NOT EXISTS (
			SELECT 1 FROM folder_members m
			WHERE m.folder_id = f.id AND m.user_id = f.owner_id AND m.deleted_at IS NULL
		  )
	`).Error
}

func ensureFolderIndexes(db *gorm.DB) error {
	for _, stmt := range []string{
		`DROP INDEX IF EXISTS idx_folder_scope`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_folders_path ON folders (path) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_folders_slug ON folders (slug) WHERE deleted_at IS NULL`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func backfillIssuePullSeqs(db *gorm.DB) error {
	for _, stmt := range []string{
		`UPDATE repositories
		SET issue_seq = (SELECT COALESCE(MAX(number), 0) FROM issues WHERE issues.repository_id = repositories.id)
		WHERE COALESCE(issue_seq, 0) = 0`,
		`UPDATE repositories
		SET pull_seq = (SELECT COALESCE(MAX(number), 0) FROM pull_requests WHERE pull_requests.repository_id = repositories.id)
		WHERE COALESCE(pull_seq, 0) = 0`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func addRepoMirrorColumns(db *gorm.DB) error {
	cols := []struct {
		name string
		sql  string
	}{
		{"mirror_direction", `ALTER TABLE repositories ADD COLUMN mirror_direction TEXT NOT NULL DEFAULT ''`},
		{"mirror_url", `ALTER TABLE repositories ADD COLUMN mirror_url TEXT NOT NULL DEFAULT ''`},
		{"mirror_username", `ALTER TABLE repositories ADD COLUMN mirror_username TEXT NOT NULL DEFAULT ''`},
		{"mirror_password", `ALTER TABLE repositories ADD COLUMN mirror_password TEXT NOT NULL DEFAULT ''`},
		{"mirror_last_sync_at", `ALTER TABLE repositories ADD COLUMN mirror_last_sync_at DATETIME`},
		{"mirror_last_error", `ALTER TABLE repositories ADD COLUMN mirror_last_error TEXT NOT NULL DEFAULT ''`},
	}
	for _, col := range cols {
		if err := migrate.AddColumnIfMissing(db, "repositories", col.name, col.sql); err != nil {
			return err
		}
	}
	return nil
}
