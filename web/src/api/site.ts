import { api } from './client'

export const siteApi = {
  get() {
    return api<{ name: string }>('/api/site')
  },
}
