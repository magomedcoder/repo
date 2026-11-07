import { api } from '../client'
import type { AdminUser, UserRole } from '../types'

export const adminUsersApi = {
  list(params?: { q?: string; role?: string; sort?: string; limit?: number; offset?: number }) {
    const q = new URLSearchParams()
    if (params?.q) {
      q.set('q', params.q)
    }

    if (params?.role) {
      q.set('role', params.role)
    }

    if (params?.sort) {
      q.set('sort', params.sort)
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
  create(body: {
    username: string
    email: string
    password: string
    role: UserRole
    current_password: string
  }) {
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
