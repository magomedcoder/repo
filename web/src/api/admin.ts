import { api } from './client'
import type { AccessLog, AdminMirror, AdminUser, UserRole } from './types'

export const adminUsersApi = {
  list(params?: { q?: string; limit?: number; offset?: number }) {
    const q = new URLSearchParams()
    if (params?.q) {
      q.set('q', params.q)
    }

    if (params?.limit != null) {
      q.set('limit', String(params.limit))
    }

    if (params?.offset != null) {
      q.set('offset', String(params.offset))
    }
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
  update(id: number, body: { email?: string; role?: UserRole; password?: string; current_password?: string }) {
    return api<AdminUser>(`/api/admin/users/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(body),
    })
  },
  remove(id: number, current_password: string) {
    return api<void>(`/api/admin/users/${id}`, {
      method: 'DELETE',
      body: JSON.stringify({ current_password }),
    })
  },
}

export const adminAccessLogsApi = {
  list(params?: { action?: string; ip?: string; username?: string; limit?: number; offset?: number }) {
    const q = new URLSearchParams()
    if (params?.action) {
      q.set('action', params.action)
    }

    if (params?.ip) {
      q.set('ip', params.ip)
    }

    if (params?.username) {
      q.set('username', params.username)
    }

    if (params?.limit != null) {
      q.set('limit', String(params.limit))
    }

    if (params?.offset != null) {
      q.set('offset', String(params.offset))
    }
    const qs = q.toString()
    return api<{ items: AccessLog[]; total: number; limit: number; offset: number }>(
      `/api/admin/access-logs${qs ? `?${qs}` : ''}`,
    )
  },
}

export const adminMirrorsApi = {
  list(params?: { direction?: string; error?: boolean }) {
    const q = new URLSearchParams()
    if (params?.direction) {
      q.set('direction', params.direction)
    }
    
    if (params?.error) {
      q.set('errors', '1')
    }
    
    const qs = q.toString()
    return api<{ mirrors: AdminMirror[] }>(`/api/admin/mirrors${qs ? `?${qs}` : ''}`)
  },
  sync(ids?: number[]) {
    return api<{ queued?: number; affected?: number }>('/api/admin/mirrors/sync', {
      method: 'POST',
      body: JSON.stringify(ids?.length ? { ids, repo_ids: ids } : {}),
    })
  },
  pause(ids: number[], paused: boolean) {
    return api<{ affected?: number }>('/api/admin/mirrors/pause', {
      method: 'POST',
      body: JSON.stringify({ 
        ids,
        repo_ids: ids,
        paused,
        pause: paused
      }),
    })
  },
  importOrg(body: {
    folder_path?: string
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
        body: JSON.stringify(body)
      }
    )
  },
}
