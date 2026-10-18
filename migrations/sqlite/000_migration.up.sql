CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    username TEXT NOT NULL,
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    avatar_path TEXT
);

CREATE TABLE IF NOT EXISTS sessions (
    id INTEGER PRIMARY KEY,
    created_at DATETIME,
    token TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    expires_at DATETIME
);

CREATE TABLE IF NOT EXISTS access_tokens (
    id INTEGER PRIMARY KEY,
    created_at DATETIME,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    prefix TEXT NOT NULL,
    last_used_at DATETIME
);

CREATE TABLE IF NOT EXISTS folders (
    id INTEGER PRIMARY KEY,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    owner_kind TEXT NOT NULL DEFAULT 'user',
    owner_id INTEGER NOT NULL,
    parent_id INTEGER NOT NULL DEFAULT 0,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    path TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS repositories (
    id INTEGER PRIMARY KEY,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    name TEXT NOT NULL,
    owner_kind TEXT NOT NULL DEFAULT 'user',
    owner_id INTEGER NOT NULL,
    folder_id INTEGER NOT NULL DEFAULT 0,
    description TEXT,
    is_private NUMERIC NOT NULL DEFAULT 0,
    protect_default_branch NUMERIC NOT NULL DEFAULT 1,
    default_branch TEXT NOT NULL DEFAULT 'main',
    path TEXT NOT NULL,
    repo_path TEXT NOT NULL DEFAULT '',
    last_activity_at DATETIME
);

CREATE TABLE IF NOT EXISTS issues (
    id INTEGER PRIMARY KEY,
    created_at DATETIME,
    updated_at DATETIME,
    repository_id INTEGER NOT NULL,
    number INTEGER NOT NULL,
    title TEXT NOT NULL,
    body TEXT,
    state TEXT NOT NULL,
    author_id INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS issue_comments (
    id INTEGER PRIMARY KEY,
    created_at DATETIME,
    updated_at DATETIME,
    issue_id INTEGER NOT NULL,
    author_id INTEGER NOT NULL,
    body TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS labels (
    id INTEGER PRIMARY KEY,
    created_at DATETIME,
    updated_at DATETIME,
    repository_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    color TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS issue_labels (
    issue_id INTEGER NOT NULL,
    label_id INTEGER NOT NULL,
    PRIMARY KEY (issue_id, label_id)
);

CREATE TABLE IF NOT EXISTS pull_requests (
    id INTEGER PRIMARY KEY,
    created_at DATETIME,
    updated_at DATETIME,
    repository_id INTEGER NOT NULL,
    number INTEGER NOT NULL,
    title TEXT NOT NULL,
    body TEXT,
    state TEXT NOT NULL,
    author_id INTEGER NOT NULL,
    base_branch TEXT NOT NULL,
    head_branch TEXT NOT NULL,
    merged_at DATETIME,
    merged_by INTEGER
);

CREATE TABLE IF NOT EXISTS pull_request_comments (
    id INTEGER PRIMARY KEY,
    created_at DATETIME,
    updated_at DATETIME,
    pull_id INTEGER NOT NULL,
    author_id INTEGER NOT NULL,
    body TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS activity_events (
    id INTEGER PRIMARY KEY,
    created_at DATETIME,
    actor_id INTEGER NOT NULL,
    owner_kind TEXT NOT NULL,
    owner_id INTEGER NOT NULL,
    repo_id INTEGER,
    org_id INTEGER,
    type TEXT NOT NULL,
    summary TEXT NOT NULL,
    ref TEXT
);

DROP TABLE IF EXISTS organization_members;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS ssh_keys;

ALTER TABLE users ADD COLUMN avatar_path TEXT;
ALTER TABLE folders ADD COLUMN owner_kind TEXT NOT NULL DEFAULT 'user';
ALTER TABLE folders ADD COLUMN owner_id INTEGER NOT NULL DEFAULT 0;
ALTER TABLE folders ADD COLUMN parent_id INTEGER NOT NULL DEFAULT 0;
ALTER TABLE folders ADD COLUMN slug TEXT NOT NULL DEFAULT '';
ALTER TABLE folders ADD COLUMN path TEXT NOT NULL DEFAULT '';
ALTER TABLE repositories ADD COLUMN owner_kind TEXT NOT NULL DEFAULT 'user';
ALTER TABLE repositories ADD COLUMN folder_id INTEGER NOT NULL DEFAULT 0;
ALTER TABLE repositories ADD COLUMN is_private NUMERIC NOT NULL DEFAULT 0;
ALTER TABLE repositories ADD COLUMN protect_default_branch NUMERIC NOT NULL DEFAULT 1;
ALTER TABLE repositories ADD COLUMN default_branch TEXT NOT NULL DEFAULT 'main';
ALTER TABLE repositories ADD COLUMN repo_path TEXT NOT NULL DEFAULT '';
ALTER TABLE repositories ADD COLUMN last_activity_at DATETIME;

UPDATE repositories SET owner_kind = 'user' WHERE owner_kind IS NULL OR owner_kind = '';
UPDATE repositories SET default_branch = 'main' WHERE default_branch IS NULL OR default_branch = '';
UPDATE repositories SET repo_path = name || '.git' WHERE repo_path = '' AND folder_id = 0;
UPDATE repositories SET repo_path = (
    SELECT CASE WHEN folders.path = '' THEN repositories.name || '.git' ELSE folders.path || '/' || repositories.name || '.git' END
    FROM folders WHERE folders.id = repositories.folder_id
) WHERE repo_path = '' AND folder_id != 0;
UPDATE repositories SET repo_path = 'repo-' || id || '/' || name || '.git' WHERE repo_path = '';
UPDATE folders SET owner_kind = 'user' WHERE owner_kind IS NULL OR owner_kind = '';

DROP INDEX IF EXISTS idx_repositories_name;
DROP INDEX IF EXISTS uni_repositories_name;
DROP INDEX IF EXISTS idx_repo_scope;
DROP INDEX IF EXISTS idx_folder_scope;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users (username);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users (deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_token ON sessions (token);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions (user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_access_tokens_token_hash ON access_tokens (token_hash);
CREATE INDEX IF NOT EXISTS idx_access_tokens_user_id ON access_tokens (user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_folder_scope ON folders (owner_kind, owner_id, parent_id, slug);
CREATE INDEX IF NOT EXISTS idx_folders_path ON folders (path);
CREATE INDEX IF NOT EXISTS idx_folders_deleted_at ON folders (deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_repo_scope ON repositories (name, owner_kind, owner_id, folder_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_repositories_repo_path ON repositories (repo_path);
CREATE INDEX IF NOT EXISTS idx_repositories_deleted_at ON repositories (deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_issue_number ON issues (repository_id, number);
CREATE INDEX IF NOT EXISTS idx_issues_state ON issues (state);
CREATE INDEX IF NOT EXISTS idx_issues_author_id ON issues (author_id);
CREATE INDEX IF NOT EXISTS idx_issue_comments_issue_id ON issue_comments (issue_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_label_name ON labels (repository_id, name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_pull_number ON pull_requests (repository_id, number);
CREATE INDEX IF NOT EXISTS idx_pull_requests_state ON pull_requests (state);
CREATE INDEX IF NOT EXISTS idx_pull_requests_author_id ON pull_requests (author_id);
CREATE INDEX IF NOT EXISTS idx_pull_request_comments_pull_id ON pull_request_comments (pull_id);
CREATE INDEX IF NOT EXISTS idx_activity_events_actor_id ON activity_events (actor_id);
CREATE INDEX IF NOT EXISTS idx_activity_events_owner_kind ON activity_events (owner_kind);
CREATE INDEX IF NOT EXISTS idx_activity_events_owner_id ON activity_events (owner_id);
CREATE INDEX IF NOT EXISTS idx_activity_events_repo_id ON activity_events (repo_id);
CREATE INDEX IF NOT EXISTS idx_activity_events_org_id ON activity_events (org_id);
CREATE INDEX IF NOT EXISTS idx_activity_events_type ON activity_events (type);
