export interface User {
  id: number
  username: string
  email: string
  has_avatar?: boolean
  created_at: string
}

export interface Profile {
  id: number
  username: string
  email?: string
  has_avatar: boolean
  created_at: string
}

export interface Folder {
  id: number
  parent_id: number | null
  name: string
  slug: string
  path: string
  created_at: string
  updated_at: string
}

export interface FolderTreeNode extends Folder {
  children: FolderTreeNode[]
}

export interface RepoSummary {
  id: number
  name: string
  description: string
  folder_id: number | null
  is_private: boolean
  default_branch: string
  created_at: string
  updated_at: string
}

export interface Repository {
  id: number
  name: string
  description: string
  owner: string
  folder_id: number | null
  folder_path: string
  is_private: boolean
  default_branch: string
  last_activity_at?: string
  created_at: string
  updated_at: string
}

export interface FolderContents {
  folder?: Folder
  folders: Folder[]
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

export interface SSHKey {
  id: number
  title: string
  fingerprint: string
  created_at: string
  last_used_at?: string
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
