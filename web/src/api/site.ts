import { api } from './client'

export const siteApi = {
  get() {
    return api<{ name: string; public_url?: string }>('/api/site')
  },
}
