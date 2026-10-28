import type { Repository, RepoSummary } from '@/api/types'
import { repositoryPath } from '@/lib/repoPath'

export type ContentRow = {
  id: number
  name: string
  description: string
  folder_path: string
  owner: string
  updated_at: string
  last_activity_at?: string
  repo_path: string
}

export function toContentRow(
  repo: RepoSummary | Repository,
  folderPath?: string,
  repoOwner?: string,
): ContentRow {
  const owner = 'owner' in repo && repo.owner ? repo.owner : repoOwner || ''
  const repo_path = repositoryPath(repo)
  const folder_path = 'folder_path' in repo && repo.folder_path ? repo.folder_path : folderPath || ''
  return {
    id: repo.id,
    name: repo.name,
    description: repo.description,
    folder_path,
    owner,
    updated_at: repo.updated_at,
    last_activity_at: 'last_activity_at' in repo ? repo.last_activity_at : undefined,
    repo_path,
  }
}
