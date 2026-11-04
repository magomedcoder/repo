DROP INDEX IF EXISTS idx_folders_slug;
DROP INDEX IF EXISTS idx_folders_path;
CREATE UNIQUE INDEX IF NOT EXISTS idx_folder_scope ON folders (owner_kind, owner_id, ifnull(parent_id, 0), slug) WHERE deleted_at IS NULL;

DROP TABLE IF EXISTS repo_collaborators;
DROP TABLE IF EXISTS team_repo_grants;
DROP TABLE IF EXISTS team_members;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS folder_members;
