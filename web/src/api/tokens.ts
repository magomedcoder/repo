import { api } from './client'
import type { AccessToken, CreateAccessTokenResponse } from './types'

export const tokensApi = {
  list() {
    return api<{ tokens: AccessToken[] }>('/api/tokens')
  },
  create(body: {
    name: string
    scope?: string
    expires_in?: number
    expires_at?: string
    current_password: string
  }) {
    return api<CreateAccessTokenResponse>('/api/tokens', {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  revoke(id: number, current_password: string) {
    return api<void>(`/api/tokens/${id}`, {
      method: 'DELETE',
      body: JSON.stringify({ current_password }),
    })
  },
}
