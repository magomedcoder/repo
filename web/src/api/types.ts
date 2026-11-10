export const RoleUser = 0 as const
export const RoleAdmin = 1 as const
export type UserRole = typeof RoleUser | typeof RoleAdmin

export interface User {
  id: number
  username: string
  email: string
  role: UserRole
  has_avatar?: boolean
  created_at: string
}

export interface Profile {
  id: number
  username: string
  email?: string
  role?: UserRole
  has_avatar: boolean
  created_at: string
}

export interface AdminUser {
  id: number
  username: string
  email: string
  role: UserRole
  has_avatar: boolean
  created_at: string
}

export interface AccessLog {
  id: number
  action: string
  ip: string
  user_agent: string
  user_id?: number
  username: string
  path: string
  repo: string
  ref: string
  created_at: string
}

export interface AuthSession {
  id: number
  ip: string
  user_agent: string
  current: boolean
  created_at: string
  last_seen_at: string
  expires_at: string
}

export type RepoCollabRole = 'admin' | 'write' | 'read'

export interface Team {
  name: string
  slug: string
  created_at: string
  updated_at: string
}

export interface TeamMember {
  user_id: number
  username: string
  created_at: string
}

export interface TeamRepoGrant {
  repo_id: number
  repo_name: string
  repo_path: string
  owner?: string
  role: RepoCollabRole | string
}

export interface RepoTeamGrant {
  team_id: number
  team_name: string
  team_slug: string
  team_owner?: string
  role: RepoCollabRole | string
  owned: boolean
}

export interface RepoCollaborator {
  user_id: number
  username: string
  role: RepoCollabRole | string
  created_at: string
  updated_at: string
}

export interface RepoSummary {
  id: number
  name: string
  description: string
  is_private: boolean
  default_branch: string
  repo_path?: string
  created_at: string
  updated_at: string
}

export type MirrorDirection = 'pull' | 'push'
export type MirrorStatusKind = 'idle' | 'syncing' | 'ok' | 'error' | 'paused'
export type MirrorConflictPolicy = 'force' | 'reject'
export type MirrorSyncTrigger = 'manual' | 'schedule' | 'push' | 'merge' | 'configure' | 'bulk'
export type MirrorSyncEventStatus = 'ok' | 'failed' | 'running'

export type MirrorFields = {
  direction?: MirrorDirection | string
  status?: MirrorStatusKind | string
  url?: string
  username?: string
  interval?: string
  sync_window?: string
  force?: boolean
  prune?: boolean
  refspecs?: string
  max_bytes?: number
  conflict_policy?: MirrorConflictPolicy | string
  insecure_tls?: boolean
  fail_count?: number
  last_sync_at?: string
  last_error?: string
  next_sync_at?: string
  last_progress?: string
  remote_head?: string
  owner_only?: boolean
  has_password?: boolean
  has_ssh_key?: boolean
  enabled?: boolean
}

export interface MirrorConfigureBody {
  enabled?: boolean
  clear?: boolean
  direction?: MirrorDirection | string
  url?: string
  username?: string
  password?: string
  clear_password?: boolean
  ssh_private_key?: string
  ssh_passphrase?: string
  clear_ssh_key?: boolean
  interval?: string
  sync_window?: string
  force?: boolean
  prune?: boolean
  refspecs?: string
  max_bytes?: number
  conflict_policy?: MirrorConflictPolicy | string
  insecure_tls?: boolean
  pinned_hosts?: string
  owner_only?: boolean
  verify_signed?: boolean
  current_password?: string
}

export interface Repository {
  id: number
  name: string
  description: string
  owner: string
  repo_path?: string
  is_private: boolean
  protect_default_branch?: boolean
  issues_enabled?: boolean
  default_branch: string
  last_activity_at?: string
  mirror_direction?: MirrorDirection | string
  mirror_url?: string
  mirror_username?: string
  mirror_has_password?: boolean
  mirror_last_sync_at?: string
  mirror_last_error?: string
  mirror_enabled?: boolean
  mirror_status?: MirrorStatusKind | string
  mirror_interval?: string
  mirror_sync_window?: string
  mirror_force?: boolean
  mirror_prune?: boolean
  mirror_refspecs?: string
  mirror_verify_signed?: boolean
  mirror_max_bytes?: number
  mirror_conflict_policy?: MirrorConflictPolicy | string
  mirror_insecure_tls?: boolean
  mirror_pinned_hosts?: string
  mirror_fail_count?: number
  mirror_next_sync_at?: string
  mirror_last_progress?: string
  mirror_remote_head?: string
  mirror_has_ssh_key?: boolean
  mirror_owner_only?: boolean
  can_read?: boolean
  can_write?: boolean
  can_admin?: boolean
  created_at: string
  updated_at: string
}

export interface MirrorStatus extends MirrorFields {
  enabled: boolean
  status: MirrorStatusKind | string
}

export interface MirrorSyncEvent {
  id: number
  trigger: MirrorSyncTrigger | string
  direction?: MirrorDirection | string
  status: MirrorSyncEventStatus | string
  error?: string
  started_at: string
  finished_at?: string
  duration_ms?: number
}

export interface MirrorDryRunRef {
  name: string
  commit_sha?: string
  old_sha?: string
  action?: 'create' | 'update' | 'delete' | 'forced' | string
}

export interface MirrorDryRun {
  refs: MirrorDryRunRef[]
  head?: string
  default_branch?: string
  truncated?: boolean
}

export interface MirrorCheckResult {
  ok?: boolean
  message?: string
  head?: string
  default_branch?: string
  refs?: RefInfo[]
}

export interface AdminMirror {
  id: number
  repo_id?: number
  repo_path: string
  direction: MirrorDirection | string
  status: MirrorStatusKind | string
  url?: string
  enabled?: boolean
  paused?: boolean
  interval?: string
  fail_count?: number
  last_sync_at?: string
  last_error?: string
  next_sync_at?: string
  group_id?: number
}

export interface MirrorGroup {
  id: number
  name: string
  upstream_url?: string
  direction?: string
  interval?: string
}

export interface RefInfo {
  name: string
  commit_sha: string
}

export interface CommitInfo {
  sha: string
  message: string
  author_name: string
  author_email: string
  authored_at: string
  parents: string[]
}

export interface TreeEntry {
  name: string
  path: string
  type: 'tree' | 'blob'
  mode: string
  size: number
  sha: string
  modified_at?: string
}

export interface BlobContent {
  path: string
  size: number
  is_binary: boolean
  content?: string
  encoding?: string
}

export interface FileDiff {
  path: string
  status: string
  old_path?: string
  patch: string
}

export interface CommitDiff {
  sha: string
  message: string
  files: FileDiff[]
}

export interface RepoStats {
  commit_count: number
  languages: Record<string, number>
}

export interface RepoBrowseSummary {
  repo?: Repository
  tree: TreeEntry[]
  stats?: RepoStats | null
  readme?: BlobContent | null
}

export interface Label {
  id: number
  name: string
  color: string
}

export interface Issue {
  number: number
  title: string
  body: string
  state: 'open' | 'closed'
  author: string
  labels: Label[]
  comment_count: number
  created_at: string
  updated_at: string
}

export interface IssueComment {
  id: number
  body: string
  author: string
  created_at: string
  updated_at: string
}

export interface IssueDetail extends Issue {
  comments: IssueComment[]
}

export interface AccessToken {
  id: number
  name: string
  prefix: string
  scope: string
  repo_ids?: number[]
  created_at: string
  last_used_at?: string
  expires_at?: string
}

export interface CreateAccessTokenResponse extends AccessToken {
  token: string
}

export interface ActivityEvent {
  id: number
  type: string
  summary: string
  ref?: string
  actor: string
  owner: string
  created_at: string
}

export interface BlameResult {
  path: string
  ref: string
  lines: {
    line: number
    content: string
    commit_sha: string
    author_name: string
    authored_at: string
  }[]
}

export interface PullRequest {
  number: number
  title: string
  body: string
  state: 'open' | 'closed' | 'merged'
  author: string
  base_branch: string
  head_branch: string
  comment_count: number
  merged_at?: string
  merged_by?: string
  created_at: string
  updated_at: string
}

export interface PullDetail extends PullRequest {
  comments: IssueComment[]
  can_fast_forward: boolean
  commits: number
  files_changed: number
}

export interface PullCompare {
  base_sha: string
  head_sha: string
  can_fast_forward: boolean
  commits: CommitInfo[]
  files: FileDiff[]
}
