import { encodeRepoPath } from '@/lib/repoPath'
import { api } from './client'
import type { BlameResult, BlobContent, CommitDiff, CommitInfo, MirrorCheckResult, MirrorConfigureBody, MirrorDryRun, MirrorStatus, MirrorSyncEvent, RefInfo, RepoBrowseSummary, RepoCollaborator, RepoCollabRole, RepoTeamGrant, Repository, TreeEntry } from './types'
export type { MirrorConfigureBody } from './types'

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
  listShared(sort?: string, paging?: Pick<ListParams, 'offset' | 'limit'>) {
    const params = new URLSearchParams({ scope: 'shared' })
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
  get(path: string) {
    return api<Repository>(`/api/repos/${encodeRepoPath(path)}`)
  },
  create(body: {
    name: string
    description?: string
    private?: boolean
    show_on_home?: boolean
    default_branch?: string
    mirror_direction?: string
    mirror_url?: string
    mirror_username?: string
    mirror_password?: string
    mirror_interval?: string
  }) {
    return api<Repository>('/api/repos', {
      method: 'POST',
      body: JSON.stringify(body)
    })
  },
  mirrorStatus(path: string) {
    return api<MirrorStatus>(`/api/repos/${encodeRepoPath(path)}/mirror`)
  },
  configureMirror(path: string, body: MirrorConfigureBody) {
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
  checkMirror(path: string, body: Record<string, unknown> = {}) {
    return api<MirrorCheckResult>(`/api/repos/${encodeRepoPath(path)}/mirror/check`, {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  mirrorHistory(path: string, params?: { limit?: number }) {
    const qs = params?.limit != null ? `?limit=${params.limit}` : ''
    return api<{ events: MirrorSyncEvent[] }>(`/api/repos/${encodeRepoPath(path)}/mirror/history${qs}`)
  },
  mirrorDryRun(path: string) {
    return api<MirrorDryRun>(`/api/repos/${encodeRepoPath(path)}/mirror/dry-run`)
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
      current_password?: string
    },
  ) {
    return api<Repository>(`/api/repos/${encodeRepoPath(path)}`, {
      method: 'PATCH',
      body: JSON.stringify(body),
    })
  },
  remove(path: string, confirm: string, current_password: string) {
    return api<void>(`/api/repos/${encodeRepoPath(path)}`, {
      method: 'DELETE',
      body: JSON.stringify({ confirm, current_password }),
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
  listTeams(path: string) {
    return api<{ teams: RepoTeamGrant[] }>(`/api/repos/${encodeRepoPath(path)}/teams`)
  },
  grantTeam(path: string, body: { team: string; role: RepoCollabRole | string }) {
    return api<RepoTeamGrant>(`/api/repos/${encodeRepoPath(path)}/teams`, {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  revokeTeam(path: string, teamID: number) {
    return api<void>(`/api/repos/${encodeRepoPath(path)}/teams/${teamID}`, {
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
