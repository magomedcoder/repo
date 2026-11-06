package migrations

import (
	"gorm.io/gorm"
	"magomedcoder.ru/repo/pkg/migrate"
)

func v1() migrate.Migration {
	return migrate.Migration{
		Version:     1,
		Name:        "schema",
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
	if err := addRepoMirrorColumns(db); err != nil {
		return err
	}

	if err := backfillMirrorStatus(db); err != nil {
		return err
	}

	return createMirrorTables(db)
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
		{"mirror_enabled", `ALTER TABLE repositories ADD COLUMN mirror_enabled NUMERIC NOT NULL DEFAULT 1`},
		{"mirror_interval", `ALTER TABLE repositories ADD COLUMN mirror_interval TEXT NOT NULL DEFAULT ''`},
		{"mirror_sync_window", `ALTER TABLE repositories ADD COLUMN mirror_sync_window TEXT NOT NULL DEFAULT ''`},
		{"mirror_force", `ALTER TABLE repositories ADD COLUMN mirror_force NUMERIC NOT NULL DEFAULT 1`},
		{"mirror_prune", `ALTER TABLE repositories ADD COLUMN mirror_prune NUMERIC NOT NULL DEFAULT 1`},
		{"mirror_refspecs", `ALTER TABLE repositories ADD COLUMN mirror_refspecs TEXT NOT NULL DEFAULT ''`},
		{"mirror_ssh_private_key", `ALTER TABLE repositories ADD COLUMN mirror_ssh_private_key TEXT NOT NULL DEFAULT ''`},
		{"mirror_ssh_passphrase", `ALTER TABLE repositories ADD COLUMN mirror_ssh_passphrase TEXT NOT NULL DEFAULT ''`},
		{"mirror_insecure_tls", `ALTER TABLE repositories ADD COLUMN mirror_insecure_tls NUMERIC NOT NULL DEFAULT 0`},
		{"mirror_pinned_hosts", `ALTER TABLE repositories ADD COLUMN mirror_pinned_hosts TEXT NOT NULL DEFAULT ''`},
		{"mirror_lfs", `ALTER TABLE repositories ADD COLUMN mirror_lfs NUMERIC NOT NULL DEFAULT 0`},
		{"mirror_max_bytes", `ALTER TABLE repositories ADD COLUMN mirror_max_bytes INTEGER NOT NULL DEFAULT 0`},
		{"mirror_conflict_policy", `ALTER TABLE repositories ADD COLUMN mirror_conflict_policy TEXT NOT NULL DEFAULT ''`},
		{"mirror_status", `ALTER TABLE repositories ADD COLUMN mirror_status TEXT NOT NULL DEFAULT ''`},
		{"mirror_fail_count", `ALTER TABLE repositories ADD COLUMN mirror_fail_count INTEGER NOT NULL DEFAULT 0`},
		{"mirror_next_sync_at", `ALTER TABLE repositories ADD COLUMN mirror_next_sync_at DATETIME`},
		{"mirror_last_progress", `ALTER TABLE repositories ADD COLUMN mirror_last_progress TEXT NOT NULL DEFAULT ''`},
		{"mirror_remote_head", `ALTER TABLE repositories ADD COLUMN mirror_remote_head TEXT NOT NULL DEFAULT ''`},
		{"mirror_owner_only", `ALTER TABLE repositories ADD COLUMN mirror_owner_only NUMERIC NOT NULL DEFAULT 0`},
		{"mirror_group_id", `ALTER TABLE repositories ADD COLUMN mirror_group_id INTEGER NOT NULL DEFAULT 0`},
		{"mirror_partial", `ALTER TABLE repositories ADD COLUMN mirror_partial NUMERIC NOT NULL DEFAULT 0`},
		{"mirror_verify_signed", `ALTER TABLE repositories ADD COLUMN mirror_verify_signed NUMERIC NOT NULL DEFAULT 0`},
	}
	for _, col := range cols {
		if err := migrate.AddColumnIfMissing(db, "repositories", col.name, col.sql); err != nil {
			return err
		}
	}
	return nil
}

func backfillMirrorStatus(db *gorm.DB) error {
	return db.Exec(`
		UPDATE repositories
		SET mirror_status = CASE
			WHEN mirror_last_error != '' THEN 'error'
			WHEN mirror_last_sync_at IS NOT NULL THEN 'ok'
			ELSE 'idle'
		END
		WHERE mirror_direction != '' AND mirror_status = ''
	`).Error
}

func createMirrorTables(db *gorm.DB) error {
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS mirror_sync_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id INTEGER NOT NULL,
			direction TEXT NOT NULL DEFAULT '',
			"trigger" TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT '',
			error TEXT NOT NULL DEFAULT '',
			started_at DATETIME NOT NULL,
			finished_at DATETIME,
			duration_ms INTEGER NOT NULL DEFAULT 0,
			actor_id INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_mirror_sync_events_repo ON mirror_sync_events (repo_id, started_at)`,
		`CREATE TABLE IF NOT EXISTS mirror_targets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME,
			updated_at DATETIME,
			repo_id INTEGER NOT NULL,
			url TEXT NOT NULL DEFAULT '',
			username TEXT NOT NULL DEFAULT '',
			password TEXT NOT NULL DEFAULT '',
			direction TEXT NOT NULL DEFAULT 'push',
			enabled NUMERIC NOT NULL DEFAULT 1,
			last_sync_at DATETIME,
			last_error TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_mirror_targets_repo ON mirror_targets (repo_id)`,
		`CREATE TABLE IF NOT EXISTS mirror_groups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME,
			updated_at DATETIME,
			name TEXT NOT NULL,
			upstream_url TEXT NOT NULL DEFAULT '',
			direction TEXT NOT NULL DEFAULT 'pull',
			sync_interval TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_repositories_mirror ON repositories (mirror_direction, mirror_status) WHERE mirror_direction != ''`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}
