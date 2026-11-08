import { api } from '../client'
import type { AdminMirror, MirrorGroup } from '../types'

export type MirrorHealth = {
  in_error: number
  older_than: string
  healthy: boolean
}

export type MirrorVerifyResult = {
  results: { repo_path?: string; ok: boolean; error?: string }[]
  checked: number
  failed: number
}

export const adminMirrorsApi = {
  list(params?: { direction?: string; error?: boolean; status?: string; q?: string; group_id?: number }) {
    const q = new URLSearchParams()
    if (params?.direction) {
      q.set('direction', params.direction)
    }

    if (params?.error) {
      q.set('errors', '1')
    }

    if (params?.status) {
      q.set('status', params.status)
    }

    if (params?.q) {
      q.set('q', params.q)
    }

    if (params?.group_id != null) {
      q.set('group_id', String(params.group_id))
    }
    const qs = q.toString()
    return api<{ mirrors: AdminMirror[] }>(`/api/admin/mirrors${qs ? `?${qs}` : ''}`)
  },
  sync(ids?: number[]) {
    return api<{ queued?: number; affected?: number }>('/api/admin/mirrors/sync', {
      method: 'POST',
      body: JSON.stringify(ids?.length ? { repo_ids: ids } : {}),
    })
  },
  pause(ids: number[], paused: boolean) {
    return api<{ affected?: number }>('/api/admin/mirrors/pause', {
      method: 'POST',
      body: JSON.stringify({
        repo_ids: ids,
        paused,
      }),
    })
  },
  verify(params?: { direction?: string; status?: string; q?: string }) {
    const q = new URLSearchParams()
    if (params?.direction) {
      q.set('direction', params.direction)
    }

    if (params?.status) {
      q.set('status', params.status)
    }

    if (params?.q) {
      q.set('q', params.q)
    }
    const qs = q.toString()
    return api<MirrorVerifyResult>(`/api/admin/mirrors/verify${qs ? `?${qs}` : ''}`, {
      method: 'POST',
      body: '{}',
    })
  },
  health(olderThanSecs?: number) {
    const q = new URLSearchParams()
    if (olderThanSecs != null) {
      q.set('older_than', String(olderThanSecs))
    }
    
    const qs = q.toString()
    return api<MirrorHealth>(`/api/admin/mirrors/health${qs ? `?${qs}` : ''}`)
  },
  importOrg(body: {
    urls: string[]
    username?: string
    password?: string
    private?: boolean
    group_id?: number
    interval?: string
  }) {
    return api<{ results: { url: string; name?: string; created: boolean; error?: string }[] }>(
      '/api/admin/mirrors/import',
      {
        method: 'POST',
        body: JSON.stringify(body),
      },
    )
  },
  listGroups() {
    return api<{ groups: MirrorGroup[] }>('/api/admin/mirror-groups')
  },
  createGroup(body: { name: string; upstream_url?: string; direction?: string; interval?: string }) {
    return api<MirrorGroup>('/api/admin/mirror-groups', {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  deleteGroup(id: number) {
    return api<void>(`/api/admin/mirror-groups/${id}`, { method: 'DELETE' })
  },
}
