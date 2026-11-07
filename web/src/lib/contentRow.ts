import type { Repository, RepoSummary } from '@/api/types'
import { repositoryPath } from '@/lib/repoPath'

export type ContentRow = {
  id: number
  name: string
  description: string
  owner: string
  updated_at: string
  last_activity_at?: string
  repo_path: string
}

export function toContentRow(
  repo: RepoSummary | Repository,
  repoOwner?: string,
): ContentRow {
  const owner = 'owner' in repo && repo.owner ? repo.owner : repoOwner || ''
  const repo_path = repositoryPath(repo)
  return {
    id: repo.id,
    name: repo.name,
    description: repo.description,
    owner,
    updated_at: repo.updated_at,
    last_activity_at: 'last_activity_at' in repo ? repo.last_activity_at : undefined,
    repo_path,
  }
}
