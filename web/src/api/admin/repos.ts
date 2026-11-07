import { api } from '../client'
import type { AdminRepo } from '../types/admin'

export const adminReposApi = {
  list(params?: {
    q?: string
    private?: boolean
    mirror?: boolean
    sort?: string
    limit?: number
    offset?: number
  }) {
    const q = new URLSearchParams()
    if (params?.q) {
      q.set('q', params.q)
    }

    if (params?.private === true) {
      q.set('private', '1')
    }

    if (params?.private === false) {
      q.set('private', '0')
    }

    if (params?.mirror) {
      q.set('mirror', '1')
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
    return api<{ repos: AdminRepo[]; total: number; limit: number; offset: number }>(
      `/api/admin/repos${qs ? `?${qs}` : ''}`,
    )
  },
}
