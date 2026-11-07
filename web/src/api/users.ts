import { api } from './client'
import type { Profile } from './types'

export const usersApi = {
  get(username: string) {
    return api<Profile>(`/api/users/${username}`)
  },
  updateMe(body: { email: string; current_password: string }) {
    return api<Profile>('/api/users/me', {
      method: 'PATCH',
      body: JSON.stringify(body),
    })
  },
  uploadAvatar(file: File) {
    const body = new FormData()
    body.append('avatar', file)
    return api<Profile>('/api/users/me/avatar', {
      method: 'POST',
      body,
    })
  },
  deleteAvatar() {
    return api<Profile>('/api/users/me/avatar', {
      method: 'DELETE',
    })
  },
  avatarURL(username: string) {
    return `/api/users/${encodeURIComponent(username)}/avatar`
  },
}
