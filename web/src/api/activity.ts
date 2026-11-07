import { api } from './client'
import type { ActivityEvent } from './types'

export const activityApi = {
  listOwner(owner: string, limit = 30) {
    const q = new URLSearchParams({ limit: String(limit) })
    return api<{ events: ActivityEvent[] }>(`/api/activity/${encodeURIComponent(owner)}?${q}`)
  },
}
