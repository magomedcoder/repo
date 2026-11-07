import { encodeRepoPath } from '@/lib/repoPath'
import { api } from './client'
import type { IssueComment, PullCompare, PullDetail, PullRequest } from './types'

export const pullsApi = {
  list(path: string, state = 'open', offset = 0, limit = 30) {
    const q = new URLSearchParams({ state, offset: String(offset), limit: String(limit) })
    return api<{ pulls: PullRequest[] }>(`/api/repos/${encodeRepoPath(path)}/pulls?${q}`)
  },
  get(path: string, number: number) {
    return api<PullDetail>(`/api/repos/${encodeRepoPath(path)}/pulls/${number}`)
  },
  create(path: string, body: { title: string; body?: string; base_branch?: string; head_branch: string }) {
    return api<PullRequest>(`/api/repos/${encodeRepoPath(path)}/pulls`, { 
      method: 'POST', 
      body: JSON.stringify(body) 
    })
  },
  update(path: string, number: number, body: { title?: string; body?: string; state?: string; base_branch?: string; head_branch?: string }) {
    return api<PullRequest>(`/api/repos/${encodeRepoPath(path)}/pulls/${number}`, { 
      method: 'PATCH', 
      body: JSON.stringify(body) 
    })
  },
  remove(path: string, number: number) {
    return api<void>(`/api/repos/${encodeRepoPath(path)}/pulls/${number}`, { method: 'DELETE' })
  },
  diff(path: string, number: number) {
    return api<PullCompare>(`/api/repos/${encodeRepoPath(path)}/pulls/${number}/diff`)
  },
  merge(path: string, number: number, strategy = 'merge') {
    return api<PullRequest>(`/api/repos/${encodeRepoPath(path)}/pulls/${number}/merge`, {
      method: 'POST',
      body: JSON.stringify({ strategy }),
    })
  },
  comment(path: string, number: number, body: string) {
    return api<IssueComment>(`/api/repos/${encodeRepoPath(path)}/pulls/${number}/comments`, {
      method: 'POST',
      body: JSON.stringify({ body }),
    })
  },
  updateComment(path: string, number: number, id: number, body: string) {
    return api<IssueComment>(`/api/repos/${encodeRepoPath(path)}/pulls/${number}/comments/${id}`, {
      method: 'PATCH',
      body: JSON.stringify({ body }),
    })
  },
  removeComment(path: string, number: number, id: number) {
    return api<void>(`/api/repos/${encodeRepoPath(path)}/pulls/${number}/comments/${id}`, { method: 'DELETE' })
  },
}
