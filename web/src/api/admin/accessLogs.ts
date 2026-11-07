import { api } from '../client'
import type { AccessLog } from '../types'

export const adminAccessLogsApi = {
  list(params?: {
    action?: string
    ip?: string
    username?: string
    from?: string
    to?: string
    limit?: number
    offset?: number
  }) {
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

    if (params?.from) {
      q.set('from', params.from)
    }

    if (params?.to) {
      q.set('to', params.to)
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
