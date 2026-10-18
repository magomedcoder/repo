import { api } from './client'
import type { AccessToken, ActivityEvent, BlobContent, BlameResult, CommitDiff, CommitInfo, CreateAccessTokenResponse, Folder, FolderContents, FolderTreeNode, Issue, IssueComment, IssueDetail, Label, Profile, PullCompare, PullDetail, PullRequest, RefInfo, RepoStats, Repository, RepoSummary, TreeEntry, User } from './types'

export const authApi = {
  register(body: {
    username: string
    email: string
    password: string
  }) {
    return api<User>('/api/auth/register', {
      method: 'POST',
      body: JSON.stringify(body)
    })
  },
  login(body: {
    login: string
    password: string
  }) {
    return api<User>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify(body)
    })
  },
  me() {
    return api<User>('/api/auth/me')
  },
  logout() {
    return api<void>('/api/auth/logout', {
      method: 'POST'
    })
  },
  password(body: { current: string; next: string }) {
    return api<void>('/api/auth/password', {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
}

export const foldersApi = {
  list(parentId?: number | null) {
    const params = new URLSearchParams()
    if (parentId != null) {
      params.set('parent_id', String(parentId))
    }

    const qs = params.toString()
    return api<{
      folders: Folder[]
    }>(`/api/folders${qs ? `?${qs}` : ''}`)
  },
  tree() {
    return api<{
      tree: FolderTreeNode[]
    }>('/api/folders?tree=1')
  },
  get(id: number) {
    return api<FolderContents>(`/api/folders/${id}`)
  },
  create(body: {
    name: string
    parent_id?: number | null
    path?: string
  }) {
    return api<Folder>('/api/folders', {
      method: 'POST',
      body: JSON.stringify(body)
    })
  },
}

export const activityApi = {
  listMine(limit = 30) {
    const q = new URLSearchParams({ limit: String(limit) })
    return api<{ events: ActivityEvent[] }>(`/api/activity?${q}`)
  },
  listOwner(owner: string, limit = 30) {
    const q = new URLSearchParams({ limit: String(limit) })
    return api<{ events: ActivityEvent[] }>(`/api/activity/${encodeURIComponent(owner)}?${q}`)
  },
}

export const tokensApi = {
  list() {
    return api<{ tokens: AccessToken[] }>('/api/tokens')
  },
  create(body: { name: string }) {
    return api<CreateAccessTokenResponse>('/api/tokens', {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  revoke(id: number) {
    return api<void>(`/api/tokens/${id}`, { method: 'DELETE' })
  },
}

export const usersApi = {
  get(username: string) {
    return api<Profile>(`/api/users/${username}`)
  },
  updateMe(body: { email: string }) {
    return api<Profile>('/api/users/me', {
      method: 'PATCH',
      body: JSON.stringify(body),
    })
  },
  uploadAvatar(file: File) {
    const body = new FormData()
    body.append('avatar', file)
    return api<Profile>('/api/users/me/avatar', {
      method: 'POST',
      body,
    })
  },
  deleteAvatar() {
    return api<Profile>('/api/users/me/avatar', {
      method: 'DELETE',
    })
  },
  avatarURL(username: string) {
    return `/api/users/${encodeURIComponent(username)}/avatar`
  },
}

export const reposApi = {
  listOwn(sort?: string) {
    const params = new URLSearchParams()
    if (sort) {
      params.set('sort', sort)
    }
    const qs = params.toString()
    return api<{ repos: Repository[] }>(`/api/repos${qs ? `?${qs}` : ''}`)
  },
  search(q: string, scope?: 'public') {
    const params = new URLSearchParams({ q })
    if (scope) params.set('scope', scope)
    return api<{ repos: Repository[] }>(`/api/repos?${params}`)
  },
  get(owner: string, path: string) {
    return api<Repository>(`/api/repos/${owner}/${path}`)
  },
  create(body: {
    name?: string
    path?: string
    description?: string
    folder_id?: number | null
    private?: boolean
    default_branch?: string
  }) {
    return api<Repository>('/api/repos', {
      method: 'POST',
      body: JSON.stringify(body)
    })
  },
  update(
    owner: string,
    path: string,
    body: {
      description?: string
      private?: boolean
      default_branch?: string
      protect_default_branch?: boolean
    },
  ) {
    return api<Repository>(`/api/repos/${owner}/${path}`, {
      method: 'PATCH',
      body: JSON.stringify(body),
    })
  },
  move(owner: string, path: string, folder_id: number | null) {
    return api<Repository>(`/api/repos/${owner}/${path}/move`, {
      method: 'POST',
      body: JSON.stringify({ folder_id }),
    })
  },
  remove(owner: string, path: string) {
    return api<void>(`/api/repos/${owner}/${path}`, {
      method: 'DELETE'
    })
  },
  branches(owner: string, path: string) {
    return api<{ branches: RefInfo[] }>(`/api/repos/${owner}/${path}/branches`)
  },
  tags(owner: string, path: string) {
    return api<{ tags: RefInfo[] }>(`/api/repos/${owner}/${path}/tags`)
  },
  commits(owner: string, path: string, params: {
    ref?: string
    offset?: number
    limit?: number
  } = {}) {
    const q = new URLSearchParams()
    if (params.ref) {
      q.set('ref', params.ref)
    }

    if (params.offset != null) {
      q.set('offset', String(params.offset))
    }

    if (params.limit != null) {
      q.set('limit', String(params.limit))
    }
    
    const qs = q.toString()
    return api<{ commits: CommitInfo[] }>(`/api/repos/${owner}/${path}/commits${qs ? `?${qs}` : ''}`)
  },
  commit(owner: string, path: string, sha: string) {
    return api<CommitInfo>(`/api/repos/${owner}/${path}/commits/${sha}`)
  },
  diff(owner: string, path: string, sha: string) {
    return api<CommitDiff>(`/api/repos/${owner}/${path}/commits/${sha}/diff`)
  },
  tree(owner: string, path: string, params: { ref?: string; path?: string } = {}) {
    const q = new URLSearchParams()
    if (params.ref) {
      q.set('ref', params.ref)
    }

    if (params.path != null) {
      q.set('path', params.path)
    }

    const qs = q.toString()
    return api<{ ref: string; path: string; tree: TreeEntry[] }>(`/api/repos/${owner}/${path}/tree${qs ? `?${qs}` : ''}`)
  },
  blob(owner: string, path: string, params: { ref: string; path: string }) {
    const q = new URLSearchParams({ ref: params.ref, path: params.path })
    return api<BlobContent>(`/api/repos/${owner}/${path}/blob?${q}`)
  },
  blame(owner: string, path: string, params: { ref: string; path: string }) {
    const q = new URLSearchParams({ ref: params.ref, path: params.path })
    return api<BlameResult>(`/api/repos/${owner}/${path}/blame?${q}`)
  },
  readme(owner: string, path: string, ref?: string) {
    const q = ref ? `?ref=${encodeURIComponent(ref)}` : ''
    return api<BlobContent>(`/api/repos/${owner}/${path}/readme${q}`)
  },
  stats(owner: string, path: string, ref?: string) {
    const q = ref ? `?ref=${encodeURIComponent(ref)}` : ''
    return api<RepoStats>(`/api/repos/${owner}/${path}/stats${q}`)
  },
}

export const issuesApi = {
  list(owner: string, path: string, state = 'open', offset = 0, limit = 30) {
    const q = new URLSearchParams({
      state,
      offset: String(offset),
      limit: String(limit)
    })
    return api<{ issues: Issue[] }>(`/api/repos/${owner}/${path}/issues?${q}`)
  },
  get(owner: string, path: string, number: number) {
    return api<IssueDetail>(`/api/repos/${owner}/${path}/issues/${number}`)
  },
  create(owner: string, path: string, body: { title: string; body?: string; label_ids?: number[] }) {
    return api<Issue>(`/api/repos/${owner}/${path}/issues`, { method: 'POST', body: JSON.stringify(body) })
  },
  update(owner: string, path: string, number: number, body: { title?: string; body?: string; state?: string; label_ids?: number[] }) {
    return api<Issue>(`/api/repos/${owner}/${path}/issues/${number}`, { method: 'PATCH', body: JSON.stringify(body) })
  },
  remove(owner: string, path: string, number: number) {
    return api<void>(`/api/repos/${owner}/${path}/issues/${number}`, { method: 'DELETE' })
  },
  comment(owner: string, path: string, number: number, body: string) {
    return api<IssueComment>(`/api/repos/${owner}/${path}/issues/${number}/comments`, {
      method: 'POST',
      body: JSON.stringify({ body }),
    })
  },
  updateComment(owner: string, path: string, number: number, id: number, body: string) {
    return api<IssueComment>(`/api/repos/${owner}/${path}/issues/${number}/comments/${id}`, {
      method: 'PATCH',
      body: JSON.stringify({ body }),
    })
  },
  removeComment(owner: string, path: string, number: number, id: number) {
    return api<void>(`/api/repos/${owner}/${path}/issues/${number}/comments/${id}`, { method: 'DELETE' })
  },
  labels(owner: string, path: string) {
    return api<{ labels: Label[] }>(`/api/repos/${owner}/${path}/labels`)
  },
  createLabel(owner: string, path: string, body: { name: string; color?: string }) {
    return api<Label>(`/api/repos/${owner}/${path}/labels`, { method: 'POST', body: JSON.stringify(body) })
  },
  removeLabel(owner: string, path: string, id: number) {
    return api<void>(`/api/repos/${owner}/${path}/labels/${id}`, { method: 'DELETE' })
  },
}

export function repoFullPath(repo: Pick<Repository, 'folder_path' | 'name'> | Pick<RepoSummary, 'name'> & { folder_path?: string }) {
  const folderPath = 'folder_path' in repo ? repo.folder_path || '' : ''
  return folderPath ? `${folderPath}/${repo.name}` : repo.name
}

export const pullsApi = {
  list(owner: string, path: string, state = 'open', offset = 0, limit = 30) {
    const q = new URLSearchParams({ state, offset: String(offset), limit: String(limit) })
    return api<{ pulls: PullRequest[] }>(`/api/repos/${owner}/${path}/pulls?${q}`)
  },
  get(owner: string, path: string, number: number) {
    return api<PullDetail>(`/api/repos/${owner}/${path}/pulls/${number}`)
  },
  create(owner: string, path: string, body: { title: string; body?: string; base_branch?: string; head_branch: string }) {
    return api<PullRequest>(`/api/repos/${owner}/${path}/pulls`, { method: 'POST', body: JSON.stringify(body) })
  },
  update(owner: string, path: string, number: number, body: { title?: string; body?: string; state?: string; base_branch?: string; head_branch?: string }) {
    return api<PullRequest>(`/api/repos/${owner}/${path}/pulls/${number}`, { method: 'PATCH', body: JSON.stringify(body) })
  },
  remove(owner: string, path: string, number: number) {
    return api<void>(`/api/repos/${owner}/${path}/pulls/${number}`, { method: 'DELETE' })
  },
  diff(owner: string, path: string, number: number) {
    return api<PullCompare>(`/api/repos/${owner}/${path}/pulls/${number}/diff`)
  },
  commits(owner: string, path: string, number: number) {
    return api<{ commits: CommitInfo[] }>(`/api/repos/${owner}/${path}/pulls/${number}/commits`)
  },
  merge(owner: string, path: string, number: number, strategy = 'merge') {
    return api<PullRequest>(`/api/repos/${owner}/${path}/pulls/${number}/merge`, {
      method: 'POST',
      body: JSON.stringify({ strategy }),
    })
  },
  comment(owner: string, path: string, number: number, body: string) {
    return api<IssueComment>(`/api/repos/${owner}/${path}/pulls/${number}/comments`, {
      method: 'POST',
      body: JSON.stringify({ body }),
    })
  },
  updateComment(owner: string, path: string, number: number, id: number, body: string) {
    return api<IssueComment>(`/api/repos/${owner}/${path}/pulls/${number}/comments/${id}`, {
      method: 'PATCH',
      body: JSON.stringify({ body }),
    })
  },
  removeComment(owner: string, path: string, number: number, id: number) {
    return api<void>(`/api/repos/${owner}/${path}/pulls/${number}/comments/${id}`, { method: 'DELETE' })
  },
}

export function cloneUrl(fullPath: string) {
  const path = fullPath.replace(/\.git$/, '')
  if (import.meta.env.DEV) {
    return `http://127.0.0.1:8080/${path}.git`
  }

  return `${window.location.origin}/${path}.git`
}
