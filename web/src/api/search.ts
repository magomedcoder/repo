import { api } from './client'
import type { Repository } from './types'

export type SearchType = 'repositories' | 'code'

export interface CodeSearchHit {
  repo_name: string
  repo_path: string
  owner: string
  path: string
  line: number
  content: string
}

export const searchApi = {
  repositories(q: string, paging?: { offset?: number; limit?: number }, init?: RequestInit) {
    const params = new URLSearchParams({ q, type: 'repositories' })
    if (paging?.offset != null) {
      params.set('offset', String(paging.offset))
    }

    if (paging?.limit != null) {
      params.set('limit', String(paging.limit))
    }
    
    return api<{ type: 'repositories'; repos: Repository[] }>(`/api/search?${params}`, init)
  },
  code(q: string, paging?: { cursor?: string }, init?: RequestInit) {
    const params = new URLSearchParams({ q, type: 'code' })
    if (paging?.cursor) {
      params.set('cursor', paging.cursor)
    }
    return api<{ type: 'code'; items: CodeSearchHit[]; next_cursor?: string }>(`/api/search?${params}`, init)
  },
}
