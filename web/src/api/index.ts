import { encodeRepoPath } from '@/lib/repoPath'
import { api } from './client'
import type { AccessLog, AccessToken, ActivityEvent, AdminUser, AuthSession, BlobContent, BlameResult, CommitDiff, CommitInfo, CreateAccessTokenResponse, Folder, FolderCaps, FolderContents, FolderMember, Issue, IssueComment, IssueDetail, Label, Profile, PullCompare, PullDetail, PullRequest, RefInfo, RepoBrowseSummary, RepoCollaborator, RepoCollabRole, Repository, RepoSummary, Team, TeamMember, TeamRepoGrant, TreeEntry, User, UserRole } from './types'

export const siteApi = {
  get() {
    return api<{ name: string }>('/api/site')
  },
}

export const accessApi = {
  visit(path: string) {
    return api<void>('/api/access/visit', {
      method: 'POST',
      body: JSON.stringify({ path }),
    }).catch(() => undefined)
  },
}

export const authApi = {
  registration() {
    return api<{ allow_registration: boolean }>('/api/auth/registration')
  },
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
  listSessions() {
    return api<{ sessions: AuthSession[] }>('/api/auth/sessions')
  },
  revokeSession(id: number) {
    return api<void>(`/api/auth/sessions/${id}`, { method: 'DELETE' })
  },
  revokeOtherSessions() {
    return api<void>('/api/auth/sessions/revoke-others', { method: 'POST' })
  },
  revokeAllSessions() {
    return api<void>('/api/auth/sessions/revoke-all', { method: 'POST' })
  },
}

export const foldersApi = {
  list() {
    return api<{
      folders: Folder[]
    }>('/api/folders')
  },
  get(path: string) {
    return api<FolderContents>(`/api/folders/${encodeRepoPath(path)}`)
  },
  create(body: {
    name: string
  }) {
    return api<Folder>('/api/folders', {
      method: 'POST',
      body: JSON.stringify(body)
    })
  },
  rename(path: string, body: { name: string }) {
    return api<Folder>(`/api/folders/${encodeRepoPath(path)}`, {
      method: 'PATCH',
      body: JSON.stringify(body),
    })
  },
  remove(path: string) {
    return api<void>(`/api/folders/${encodeRepoPath(path)}`, {
      method: 'DELETE',
    })
  },
  listMembers(path: string) {
    return api<{ members: FolderMember[] }>(`/api/folders/${encodeRepoPath(path)}/members`)
  },
  addMember(path: string, body: { username: string; role: string; caps?: FolderCaps }) {
    return api<FolderMember>(`/api/folders/${encodeRepoPath(path)}/members`, {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  updateMember(path: string, username: string, body: { role: string; caps?: FolderCaps }) {
    return api<FolderMember>(`/api/folders/${encodeRepoPath(path)}/members/${encodeURIComponent(username)}`, {
      method: 'PATCH',
      body: JSON.stringify(body),
    })
  },
  removeMember(path: string, username: string) {
    return api<void>(`/api/folders/${encodeRepoPath(path)}/members/${encodeURIComponent(username)}`, {
      method: 'DELETE',
    })
  },
  listTeams(path: string) {
    return api<{ teams: Team[] }>(`/api/folders/${encodeRepoPath(path)}/teams`)
  },
  createTeam(path: string, body: { name: string; caps?: FolderCaps }) {
    return api<Team>(`/api/folders/${encodeRepoPath(path)}/teams`, {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  removeTeam(path: string, slug: string) {
    return api<void>(`/api/folders/${encodeRepoPath(path)}/teams/${encodeURIComponent(slug)}`, {
      method: 'DELETE',
    })
  },
  listTeamMembers(path: string, slug: string) {
    return api<{ members: TeamMember[] }>(`/api/folders/${encodeRepoPath(path)}/teams/${encodeURIComponent(slug)}/members`)
  },
  addTeamMember(path: string, slug: string, body: { username: string }) {
    return api<TeamMember>(`/api/folders/${encodeRepoPath(path)}/teams/${encodeURIComponent(slug)}/members`, {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  removeTeamMember(path: string, slug: string, username: string) {
    return api<void>(`/api/folders/${encodeRepoPath(path)}/teams/${encodeURIComponent(slug)}/members/${encodeURIComponent(username)}`, {
      method: 'DELETE',
    })
  },
  listTeamRepos(path: string, slug: string) {
    return api<{ repos: TeamRepoGrant[] }>(`/api/folders/${encodeRepoPath(path)}/teams/${encodeURIComponent(slug)}/repos`)
  },
  setTeamRepo(path: string, slug: string, body: { repo: string; role: RepoCollabRole | string }) {
    return api<TeamRepoGrant>(`/api/folders/${encodeRepoPath(path)}/teams/${encodeURIComponent(slug)}/repos`, {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  removeTeamRepo(path: string, slug: string, repo: string) {
    return api<void>(`/api/folders/${encodeRepoPath(path)}/teams/${encodeURIComponent(slug)}/repos/${encodeURIComponent(repo)}`, {
      method: 'DELETE',
    })
  },
}

export const activityApi = {
  listOwner(owner: string, limit = 30) {
    const q = new URLSearchParams({ limit: String(limit) })
    return api<{ events: ActivityEvent[] }>(`/api/activity/${encodeURIComponent(owner)}?${q}`)
  },
}

export const tokensApi = {
  list() {
    return api<{ tokens: AccessToken[] }>('/api/tokens')
  },
  create(body: { name: string; expires_in?: number; expires_at?: string }) {
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
  updateMe(body: { email: string; current_password: string }) {
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


type ListParams = {
  sort?: string
  offset?: number
  limit?: number
}

export const reposApi = {
  listOwn(sort?: string, paging?: Pick<ListParams, 'offset' | 'limit'>) {
    const params = new URLSearchParams()
    if (sort) {
      params.set('sort', sort)
    }
    if (paging?.offset != null) {
      params.set('offset', String(paging.offset))
    }
    if (paging?.limit != null) {
      params.set('limit', String(paging.limit))
    }
    const qs = params.toString()
    return api<{ repos: Repository[] }>(`/api/repos${qs ? `?${qs}` : ''}`)
  },
  listHome(sort?: string, paging?: Pick<ListParams, 'offset' | 'limit'>) {
    const params = new URLSearchParams({ scope: 'home' })
    if (sort) {
      params.set('sort', sort)
    }
    
    if (paging?.offset != null) {
      params.set('offset', String(paging.offset))
    }

    if (paging?.limit != null) {
      params.set('limit', String(paging.limit))
    }

    return api<{ repos: Repository[] }>(`/api/repos?${params}`)
  },
  search(
    q: string,
    scope?: 'public',
    paging?: Pick<ListParams, 'offset' | 'limit'>,
    init?: RequestInit,
  ) {
    const params = new URLSearchParams({ q })
    if (scope) params.set('scope', scope)
    if (paging?.offset != null) {
      params.set('offset', String(paging.offset))
    }
    if (paging?.limit != null) {
      params.set('limit', String(paging.limit))
    }
    return api<{ repos: Repository[] }>(`/api/repos?${params}`, init)
  },
  get(path: string) {
    return api<Repository>(`/api/repos/${encodeRepoPath(path)}`)
  },
  create(body: {
    name: string
    description?: string
    folder_path?: string
    private?: boolean
    show_on_home?: boolean
    default_branch?: string
    mirror_direction?: string
    mirror_url?: string
    mirror_username?: string
    mirror_password?: string
  }) {
    return api<Repository>('/api/repos', {
      method: 'POST',
      body: JSON.stringify(body)
    })
  },
  configureMirror(
    path: string,
    body: {
      enabled?: boolean
      direction?: string
      url?: string
      username?: string
      password?: string
      clear_password?: boolean
    },
  ) {
    return api<Repository>(`/api/repos/${encodeRepoPath(path)}/mirror`, {
      method: 'PUT',
      body: JSON.stringify(body),
    })
  },
  syncMirror(path: string) {
    return api<Repository>(`/api/repos/${encodeRepoPath(path)}/mirror/sync`, {
      method: 'POST',
    })
  },
  update(
    path: string,
    body: {
      name?: string
      description?: string
      private?: boolean
      show_on_home?: boolean
      default_branch?: string
      protect_default_branch?: boolean
      issues_enabled?: boolean
    },
  ) {
    return api<Repository>(`/api/repos/${encodeRepoPath(path)}`, {
      method: 'PATCH',
      body: JSON.stringify(body),
    })
  },
  move(path: string, folder_path: string) {
    return api<Repository>(`/api/repos/${encodeRepoPath(path)}/move`, {
      method: 'POST',
      body: JSON.stringify({ folder_path }),
    })
  },
  remove(path: string) {
    return api<void>(`/api/repos/${encodeRepoPath(path)}`, {
      method: 'DELETE'
    })
  },
  listCollaborators(path: string) {
    return api<{ collaborators: RepoCollaborator[] }>(`/api/repos/${encodeRepoPath(path)}/collaborators`)
  },
  addCollaborator(path: string, body: { username: string; role: RepoCollabRole | string }) {
    return api<RepoCollaborator>(`/api/repos/${encodeRepoPath(path)}/collaborators`, {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  removeCollaborator(path: string, username: string) {
    return api<void>(`/api/repos/${encodeRepoPath(path)}/collaborators/${encodeURIComponent(username)}`, {
      method: 'DELETE',
    })
  },
  branches(path: string) {
    return api<{ branches: RefInfo[] }>(`/api/repos/${encodeRepoPath(path)}/branches`)
  },
  tags(path: string) {
    return api<{ tags: RefInfo[] }>(`/api/repos/${encodeRepoPath(path)}/tags`)
  },
  commits(path: string, params: {
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
    return api<{ commits: CommitInfo[] }>(`/api/repos/${encodeRepoPath(path)}/commits${qs ? `?${qs}` : ''}`)
  },
  commit(path: string, sha: string) {
    return api<CommitInfo>(`/api/repos/${encodeRepoPath(path)}/commits/${sha}`)
  },
  diff(path: string, sha: string) {
    return api<CommitDiff>(`/api/repos/${encodeRepoPath(path)}/commits/${sha}/diff`)
  },
  tree(path: string, params: { ref?: string; path?: string } = {}) {
    const q = new URLSearchParams()
    if (params.ref) {
      q.set('ref', params.ref)
    }

    if (params.path != null) {
      q.set('path', params.path)
    }

    const qs = q.toString()
    return api<{ ref: string; path: string; tree: TreeEntry[] }>(`/api/repos/${encodeRepoPath(path)}/tree${qs ? `?${qs}` : ''}`)
  },
  blob(path: string, params: { ref: string; path: string }) {
    const q = new URLSearchParams({ ref: params.ref, path: params.path })
    return api<BlobContent>(`/api/repos/${encodeRepoPath(path)}/blob?${q}`)
  },
  blame(path: string, params: { ref: string; path: string }) {
    const q = new URLSearchParams({ ref: params.ref, path: params.path })
    return api<BlameResult>(`/api/repos/${encodeRepoPath(path)}/blame?${q}`)
  },
  rawHref(repoPath: string, params: { ref: string; path: string }) {
    const q = new URLSearchParams({ ref: params.ref, path: params.path })
    return `/api/repos/${encodeRepoPath(repoPath)}/raw?${q}`
  },
  summary(path: string, ref?: string) {
    const q = ref ? `?ref=${encodeURIComponent(ref)}` : ''
    return api<RepoBrowseSummary>(`/api/repos/${encodeRepoPath(path)}/summary${q}`)
  },
}

export const issuesApi = {
  list(path: string, state = 'open', offset = 0, limit = 30) {
    const q = new URLSearchParams({
      state,
      offset: String(offset),
      limit: String(limit)
    })
    return api<{ issues: Issue[] }>(`/api/repos/${encodeRepoPath(path)}/issues?${q}`)
  },
  get(path: string, number: number) {
    return api<IssueDetail>(`/api/repos/${encodeRepoPath(path)}/issues/${number}`)
  },
  create(path: string, body: { title: string; body?: string; label_ids?: number[] }) {
    return api<Issue>(`/api/repos/${encodeRepoPath(path)}/issues`, { method: 'POST', body: JSON.stringify(body) })
  },
  update(path: string, number: number, body: { title?: string; body?: string; state?: string; label_ids?: number[] }) {
    return api<Issue>(`/api/repos/${encodeRepoPath(path)}/issues/${number}`, { method: 'PATCH', body: JSON.stringify(body) })
  },
  remove(path: string, number: number) {
    return api<void>(`/api/repos/${encodeRepoPath(path)}/issues/${number}`, { method: 'DELETE' })
  },
  comment(path: string, number: number, body: string) {
    return api<IssueComment>(`/api/repos/${encodeRepoPath(path)}/issues/${number}/comments`, {
      method: 'POST',
      body: JSON.stringify({ body }),
    })
  },
  updateComment(path: string, number: number, id: number, body: string) {
    return api<IssueComment>(`/api/repos/${encodeRepoPath(path)}/issues/${number}/comments/${id}`, {
      method: 'PATCH',
      body: JSON.stringify({ body }),
    })
  },
  removeComment(path: string, number: number, id: number) {
    return api<void>(`/api/repos/${encodeRepoPath(path)}/issues/${number}/comments/${id}`, { method: 'DELETE' })
  },
  labels(path: string) {
    return api<{ labels: Label[] }>(`/api/repos/${encodeRepoPath(path)}/labels`)
  },
  createLabel(path: string, body: { name: string; color?: string }) {
    return api<Label>(`/api/repos/${encodeRepoPath(path)}/labels`, { method: 'POST', body: JSON.stringify(body) })
  },
  removeLabel(path: string, id: number) {
    return api<void>(`/api/repos/${encodeRepoPath(path)}/labels/${id}`, { method: 'DELETE' })
  },
}


export const pullsApi = {
  list(path: string, state = 'open', offset = 0, limit = 30) {
    const q = new URLSearchParams({ state, offset: String(offset), limit: String(limit) })
    return api<{ pulls: PullRequest[] }>(`/api/repos/${encodeRepoPath(path)}/pulls?${q}`)
  },
  get(path: string, number: number) {
    return api<PullDetail>(`/api/repos/${encodeRepoPath(path)}/pulls/${number}`)
  },
  create(path: string, body: { title: string; body?: string; base_branch?: string; head_branch: string }) {
    return api<PullRequest>(`/api/repos/${encodeRepoPath(path)}/pulls`, { method: 'POST', body: JSON.stringify(body) })
  },
  update(path: string, number: number, body: { title?: string; body?: string; state?: string; base_branch?: string; head_branch?: string }) {
    return api<PullRequest>(`/api/repos/${encodeRepoPath(path)}/pulls/${number}`, { method: 'PATCH', body: JSON.stringify(body) })
  },
  remove(path: string, number: number) {
    return api<void>(`/api/repos/${encodeRepoPath(path)}/pulls/${number}`, { method: 'DELETE' })
  },
  diff(path: string, number: number) {
    return api<PullCompare>(`/api/repos/${encodeRepoPath(path)}/pulls/${number}/diff`)
  },
  merge(path: string, number: number, strategy = 'merge') {
    return api<PullRequest>(`/api/repos/${encodeRepoPath(path)}/pulls/${number}/merge`, {
      method: 'POST',
      body: JSON.stringify({ strategy }),
    })
  },
  comment(path: string, number: number, body: string) {
    return api<IssueComment>(`/api/repos/${encodeRepoPath(path)}/pulls/${number}/comments`, {
      method: 'POST',
      body: JSON.stringify({ body }),
    })
  },
  updateComment(path: string, number: number, id: number, body: string) {
    return api<IssueComment>(`/api/repos/${encodeRepoPath(path)}/pulls/${number}/comments/${id}`, {
      method: 'PATCH',
      body: JSON.stringify({ body }),
    })
  },
  removeComment(path: string, number: number, id: number) {
    return api<void>(`/api/repos/${encodeRepoPath(path)}/pulls/${number}/comments/${id}`, { method: 'DELETE' })
  },
}

export const adminUsersApi = {
  list(params?: { q?: string; limit?: number; offset?: number }) {
    const q = new URLSearchParams()
    if (params?.q) q.set('q', params.q)
    if (params?.limit != null) q.set('limit', String(params.limit))
    if (params?.offset != null) q.set('offset', String(params.offset))
    const qs = q.toString()
    return api<{ users: AdminUser[]; total: number; limit: number; offset: number }>(
      `/api/admin/users${qs ? `?${qs}` : ''}`,
    )
  },
  create(body: { username: string; email: string; password: string; role: UserRole }) {
    return api<AdminUser>('/api/admin/users', {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  update(id: number, body: { email?: string; role?: UserRole; password?: string }) {
    return api<AdminUser>(`/api/admin/users/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(body),
    })
  },
  remove(id: number) {
    return api<void>(`/api/admin/users/${id}`, { method: 'DELETE' })
  },
}

export const adminAccessLogsApi = {
  list(params?: { action?: string; ip?: string; username?: string; limit?: number; offset?: number }) {
    const q = new URLSearchParams()
    if (params?.action) q.set('action', params.action)
    if (params?.ip) q.set('ip', params.ip)
    if (params?.username) q.set('username', params.username)
    if (params?.limit != null) q.set('limit', String(params.limit))
    if (params?.offset != null) q.set('offset', String(params.offset))
    const qs = q.toString()
    return api<{ items: AccessLog[]; total: number; limit: number; offset: number }>(
      `/api/admin/access-logs${qs ? `?${qs}` : ''}`,
    )
  },
}

export function cloneUrl(fullPath: string) {
  const path = fullPath.replace(/\.git$/, '')
  const base = (import.meta.env.VITE_GIT_BASE as string | undefined)?.replace(/\/$/, '')
    || window.location.origin
  return `${base}/${path}.git`
}
