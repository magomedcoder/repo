import type { AccessLog, AdminMirror, AdminUser, MirrorGroup, UserRole } from '../types'

export type { AccessLog, AdminMirror, AdminUser, MirrorGroup, UserRole }

export type AdminOverview = {
  users: number
  repos: number
  mirrors_error: number
  events_24h: number
  recent_logs: AccessLog[]
}

export type AdminRepo = {
  id: number
  name: string
  description: string
  owner: string
  repo_path: string
  is_private: boolean
  show_on_home: boolean
  mirror_direction?: string
  mirror_status?: string
  updated_at: string
  last_activity_at?: string
}

export type MirrorHealth = {
  in_error: number
  older_than: string
  healthy: boolean
}
