import { encodeRepoPath } from '@/lib/repoPath'
import { api } from './client'
import type { Issue, IssueComment, IssueDetail, Label } from './types'

export const issuesApi = {
  list(path: string, state = 'open', offset = 0, limit = 30) {
    const q = new URLSearchParams({
      state,
      offset: String(offset),
      limit: String(limit)
    })
    return api<{ issues: Issue[] }>(`/api/repos/${encodeRepoPath(path)}/issues?${q}`)
  },
  get(path: string, number: number) {
    return api<IssueDetail>(`/api/repos/${encodeRepoPath(path)}/issues/${number}`)
  },
  create(path: string, body: { title: string; body?: string; label_ids?: number[] }) {
    return api<Issue>(`/api/repos/${encodeRepoPath(path)}/issues`, { 
      method: 'POST', 
      body: JSON.stringify(body) 
    })
  },
  update(path: string, number: number, body: { title?: string; body?: string; state?: string; label_ids?: number[] }) {
    return api<Issue>(`/api/repos/${encodeRepoPath(path)}/issues/${number}`, { method: 'PATCH', body: JSON.stringify(body) })
  },
  remove(path: string, number: number) {
    return api<void>(`/api/repos/${encodeRepoPath(path)}/issues/${number}`, { method: 'DELETE' })
  },
  comment(path: string, number: number, body: string) {
    return api<IssueComment>(`/api/repos/${encodeRepoPath(path)}/issues/${number}/comments`, {
      method: 'POST',
      body: JSON.stringify({ body }),
    })
  },
  updateComment(path: string, number: number, id: number, body: string) {
    return api<IssueComment>(`/api/repos/${encodeRepoPath(path)}/issues/${number}/comments/${id}`, {
      method: 'PATCH',
      body: JSON.stringify({ body }),
    })
  },
  removeComment(path: string, number: number, id: number) {
    return api<void>(`/api/repos/${encodeRepoPath(path)}/issues/${number}/comments/${id}`, { method: 'DELETE' })
  },
  labels(path: string) {
    return api<{ labels: Label[] }>(`/api/repos/${encodeRepoPath(path)}/labels`)
  },
  createLabel(path: string, body: { name: string; color?: string }) {
    return api<Label>(`/api/repos/${encodeRepoPath(path)}/labels`, { method: 'POST', body: JSON.stringify(body) })
  },
  removeLabel(path: string, id: number) {
    return api<void>(`/api/repos/${encodeRepoPath(path)}/labels/${id}`, { method: 'DELETE' })
  },
}
