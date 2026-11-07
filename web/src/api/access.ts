import { api } from './client'

export const accessApi = {
  visit(path: string) {
    return api<void>('/api/access/visit', {
      method: 'POST',
      body: JSON.stringify({ path }),
    }).catch(() => undefined)
  },
}
