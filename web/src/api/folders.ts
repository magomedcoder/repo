import { encodeRepoPath } from '@/lib/repoPath'
import { api } from './client'
import type { Folder, FolderCaps, FolderContents, FolderMember, RepoCollabRole, Team, TeamMember, TeamRepoGrant } from './types'

export const foldersApi = {
  list() {
    return api<{
      folders: Folder[]
    }>('/api/folders')
  },
  get(path: string) {
    return api<FolderContents>(`/api/folders/${encodeRepoPath(path)}`)
  },
  create(body: {
    name: string
  }) {
    return api<Folder>('/api/folders', {
      method: 'POST',
      body: JSON.stringify(body)
    })
  },
  rename(path: string, body: { name: string }) {
    return api<Folder>(`/api/folders/${encodeRepoPath(path)}`, {
      method: 'PATCH',
      body: JSON.stringify(body),
    })
  },
  remove(path: string) {
    return api<void>(`/api/folders/${encodeRepoPath(path)}`, {
      method: 'DELETE',
    })
  },
  listMembers(path: string) {
    return api<{ members: FolderMember[] }>(`/api/folders/${encodeRepoPath(path)}/members`)
  },
  addMember(path: string, body: { username: string; role: string; caps?: FolderCaps }) {
    return api<FolderMember>(`/api/folders/${encodeRepoPath(path)}/members`, {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  updateMember(path: string, username: string, body: { role: string; caps?: FolderCaps }) {
    return api<FolderMember>(`/api/folders/${encodeRepoPath(path)}/members/${encodeURIComponent(username)}`, {
      method: 'PATCH',
      body: JSON.stringify(body),
    })
  },
  removeMember(path: string, username: string) {
    return api<void>(`/api/folders/${encodeRepoPath(path)}/members/${encodeURIComponent(username)}`, {
      method: 'DELETE',
    })
  },
  listTeams(path: string) {
    return api<{ teams: Team[] }>(`/api/folders/${encodeRepoPath(path)}/teams`)
  },
  createTeam(path: string, body: { name: string; caps?: FolderCaps }) {
    return api<Team>(`/api/folders/${encodeRepoPath(path)}/teams`, {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  removeTeam(path: string, slug: string) {
    return api<void>(`/api/folders/${encodeRepoPath(path)}/teams/${encodeURIComponent(slug)}`, {
      method: 'DELETE',
    })
  },
  listTeamMembers(path: string, slug: string) {
    return api<{ members: TeamMember[] }>(`/api/folders/${encodeRepoPath(path)}/teams/${encodeURIComponent(slug)}/members`)
  },
  addTeamMember(path: string, slug: string, body: { username: string }) {
    return api<TeamMember>(`/api/folders/${encodeRepoPath(path)}/teams/${encodeURIComponent(slug)}/members`, {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  removeTeamMember(path: string, slug: string, username: string) {
    return api<void>(`/api/folders/${encodeRepoPath(path)}/teams/${encodeURIComponent(slug)}/members/${encodeURIComponent(username)}`, {
      method: 'DELETE',
    })
  },
  listTeamRepos(path: string, slug: string) {
    return api<{ repos: TeamRepoGrant[] }>(`/api/folders/${encodeRepoPath(path)}/teams/${encodeURIComponent(slug)}/repos`)
  },
  setTeamRepo(path: string, slug: string, body: { repo: string; role: RepoCollabRole | string }) {
    return api<TeamRepoGrant>(`/api/folders/${encodeRepoPath(path)}/teams/${encodeURIComponent(slug)}/repos`, {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  removeTeamRepo(path: string, slug: string, repo: string) {
    return api<void>(`/api/folders/${encodeRepoPath(path)}/teams/${encodeURIComponent(slug)}/repos/${encodeURIComponent(repo)}`, {
      method: 'DELETE',
    })
  },
}
