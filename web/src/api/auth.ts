import { api } from './client'
import type { AuthSession, User } from './types'

export const authApi = {
  registration() {
    return api<{ allow_registration: boolean }>('/api/auth/registration')
  },
  register(body: {
    username: string
    email: string
    password: string
  }) {
    return api<User>('/api/auth/register', {
      method: 'POST',
      body: JSON.stringify(body)
    })
  },
  login(body: {
    login: string
    password: string
  }) {
    return api<User>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify(body)
    })
  },
  me() {
    return api<User>('/api/auth/me')
  },
  logout() {
    return api<void>('/api/auth/logout', {
      method: 'POST'
    })
  },
  password(body: { current: string; next: string }) {
    return api<void>('/api/auth/password', {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  listSessions() {
    return api<{ sessions: AuthSession[] }>('/api/auth/sessions')
  },
  revokeSession(id: number) {
    return api<void>(`/api/auth/sessions/${id}`, { method: 'DELETE' })
  },
  revokeOtherSessions() {
    return api<void>('/api/auth/sessions/revoke-others', { method: 'POST' })
  },
  revokeAllSessions(current_password: string) {
    return api<void>('/api/auth/sessions/revoke-all', {
      method: 'POST',
      body: JSON.stringify({ current_password }),
    })
  },
}
