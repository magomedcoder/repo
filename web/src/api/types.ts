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

export type FolderCaps = {
  view?: boolean
  create_repos?: boolean
  create_folders?: boolean
  manage_members?: boolean
  manage_settings?: boolean
  delete_folder?: boolean
  delete_repos?: boolean
}

export type FolderRole = 'owner' | 'admin' | 'member'
export type RepoCollabRole = 'admin' | 'write' | 'read'

export interface Folder {
  name: string
  slug: string
  path: string
  role?: FolderRole | string
  caps?: FolderCaps
  created_at: string
  updated_at: string
}

export interface FolderMember {
  user_id: number
  username: string
  role: FolderRole
  caps: FolderCaps
  created_at: string
  updated_at: string
}

export interface Team {
  name: string
  slug: string
  caps: FolderCaps
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
  role: RepoCollabRole | string
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
  show_on_home?: boolean
  default_branch: string
  repo_path?: string
  folder_path?: string
  created_at: string
  updated_at: string
}

export interface Repository {
  id: number
  name: string
  description: string
  owner: string
  folder_path: string
  repo_path?: string
  is_private: boolean
  show_on_home?: boolean
  protect_default_branch?: boolean
  issues_enabled?: boolean
  default_branch: string
  last_activity_at?: string
  can_read?: boolean
  can_write?: boolean
  can_admin?: boolean
  created_at: string
  updated_at: string
}

export interface FolderContents {
  folder?: Folder
  repos: RepoSummary[]
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
  size_bytes: number
  languages: Record<string, number>
}

export interface RepoBrowseSummary {
  repo?: Repository
  branches: RefInfo[]
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
  owner_kind: string
  created_at: string
}

export interface BlameLine {
  line: number
  content: string
  commit_sha: string
  author_name: string
  authored_at: string
}

export interface BlameResult {
  path: string
  ref: string
  lines: BlameLine[]
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
