import { api } from './client'
import type { AccessToken, CreateAccessTokenResponse } from './types'

export const tokensApi = {
  list() {
    return api<{ tokens: AccessToken[] }>('/api/tokens')
  },
  create(body: { name: string; expires_in?: number; expires_at?: string }) {
    return api<CreateAccessTokenResponse>('/api/tokens', {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  revoke(id: number) {
    return api<void>(`/api/tokens/${id}`, { method: 'DELETE' })
  },
}
