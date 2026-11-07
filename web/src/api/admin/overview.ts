import { api } from '../client'
import type { AccessLog } from '../types'

export type AdminOverview = {
  users: number
  repos: number
  mirrors_error: number
  events_24h: number
  recent_logs: AccessLog[]
}

export const adminOverviewApi = {
  get() {
    return api<AdminOverview>('/api/admin/overview')
  },
}
