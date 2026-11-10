export type AdminRepo = {
  id: number
  name: string
  description: string
  owner: string
  repo_path: string
  is_private: boolean
  mirror_direction?: string
  mirror_status?: string
  updated_at: string
  last_activity_at?: string
}
