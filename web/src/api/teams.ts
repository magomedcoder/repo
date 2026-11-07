import { api } from './client'
import type { RepoCollabRole, Team, TeamMember, TeamRepoGrant } from './types'

const teamPath = (slug: string) => `/api/teams/${encodeURIComponent(slug)}`

export const teamsApi = {
  list() {
    return api<{ teams: Team[] }>('/api/teams')
  },
  get(slug: string) {
    return api<Team>(teamPath(slug))
  },
  create(body: { name: string }) {
    return api<Team>('/api/teams', {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  rename(slug: string, body: { name: string }) {
    return api<Team>(teamPath(slug), {
      method: 'PATCH',
      body: JSON.stringify(body),
    })
  },
  remove(slug: string) {
    return api<void>(teamPath(slug), { method: 'DELETE' })
  },
  listMembers(slug: string) {
    return api<{ members: TeamMember[] }>(`${teamPath(slug)}/members`)
  },
  addMember(slug: string, body: { username: string }) {
    return api<TeamMember>(`${teamPath(slug)}/members`, {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  removeMember(slug: string, username: string) {
    return api<void>(`${teamPath(slug)}/members/${encodeURIComponent(username)}`, { method: 'DELETE' })
  },
  listRepos(slug: string) {
    return api<{ repos: TeamRepoGrant[] }>(`${teamPath(slug)}/repos`)
  },
  setRepo(slug: string, body: { repo: string; role: RepoCollabRole | string }) {
    return api<TeamRepoGrant>(`${teamPath(slug)}/repos`, {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  removeRepo(slug: string, repo: string) {
    return api<void>(`${teamPath(slug)}/repos/${encodeURIComponent(repo)}`, { method: 'DELETE' })
  },
}
