CREATE TABLE IF NOT EXISTS folder_members (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at DATETIME,
  updated_at DATETIME,
  deleted_at DATETIME,
  folder_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  role TEXT NOT NULL,
  caps INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_folder_members_folder_user ON folder_members (folder_id, user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_folder_members_user ON folder_members (user_id) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS teams (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at DATETIME,
  updated_at DATETIME,
  deleted_at DATETIME,
  folder_id INTEGER NOT NULL,
  name TEXT NOT NULL,
  slug TEXT NOT NULL,
  caps INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_teams_folder_slug ON teams (folder_id, slug) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS team_members (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at DATETIME,
  deleted_at DATETIME,
  team_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_team_members_team_user ON team_members (team_id, user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_team_members_user ON team_members (user_id) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS team_repo_grants (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at DATETIME,
  updated_at DATETIME,
  deleted_at DATETIME,
  team_id INTEGER NOT NULL,
  repo_id INTEGER NOT NULL,
  role TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_team_repo_grants_team_repo ON team_repo_grants (team_id, repo_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_team_repo_grants_repo ON team_repo_grants (repo_id) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS repo_collaborators (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at DATETIME,
  updated_at DATETIME,
  deleted_at DATETIME,
  repo_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  role TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_repo_collaborators_repo_user ON repo_collaborators (repo_id, user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_repo_collaborators_user ON repo_collaborators (user_id) WHERE deleted_at IS NULL;

INSERT INTO folder_members (created_at, updated_at, folder_id, user_id, role, caps)
SELECT CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, f.id, f.owner_id, 'owner', 127
FROM folders f
WHERE f.deleted_at IS NULL
  AND f.owner_id IS NOT NULL
  AND f.owner_id > 0
  AND NOT EXISTS (
    SELECT 1 FROM folder_members m
    WHERE m.folder_id = f.id AND m.user_id = f.owner_id AND m.deleted_at IS NULL
  );

DROP INDEX IF EXISTS idx_folder_scope;
CREATE UNIQUE INDEX IF NOT EXISTS idx_folders_path ON folders (path) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_folders_slug ON folders (slug) WHERE deleted_at IS NULL AND parent_id IS NULL;

UPDATE repositories
SET issue_seq = (SELECT COALESCE(MAX(number), 0) FROM issues WHERE issues.repository_id = repositories.id)
WHERE COALESCE(issue_seq, 0) = 0;

UPDATE repositories
SET pull_seq = (SELECT COALESCE(MAX(number), 0) FROM pull_requests WHERE pull_requests.repository_id = repositories.id)
WHERE COALESCE(pull_seq, 0) = 0;
