import type { Repository, RepoSummary } from '@/api/types'

export type ContentRow = {
  id: number
  name: string
  description: string
  folder_path: string
  folder_id: number | null
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
  let repo_path = ''
  if ('repo_path' in repo && repo.repo_path) {
    repo_path = repo.repo_path.replace(/\.git$/, '')
  } else if ('folder_path' in repo && repo.folder_path) {
    repo_path = `${repo.folder_path}/${repo.name}`
  } else if (folderPath) {
    repo_path = `${folderPath}/${repo.name}`
  } else {
    repo_path = owner ? `${owner}/${repo.name}` : repo.name
  }
  
  const folder_path = 'folder_path' in repo && repo.folder_path ? repo.folder_path : folderPath || ''
  return {
    id: repo.id,
    name: repo.name,
    description: repo.description,
    folder_path,
    folder_id: 'folder_id' in repo ? repo.folder_id : null,
    owner,
    updated_at: repo.updated_at,
    last_activity_at: 'last_activity_at' in repo ? repo.last_activity_at : undefined,
    repo_path,
  }
}
