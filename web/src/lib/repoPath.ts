import type { Repository, RepoSummary } from '@/api/types'

export function normalizeRepoPath(path: string): string {
  return path.replace(/^\/+|\/+$/g, '').replace(/\.git$/i, '')
}

export function encodeRepoPath(path: string): string {
  return normalizeRepoPath(path)
    .split('/')
    .filter(Boolean)
    .map(encodeURIComponent)
    .join('/')
}

export function repositoryPath(repo: Pick<Repository, 'name' | 'owner' | 'folder_path' | 'repo_path'> | RepoSummary & { owner?: string; folder_path?: string; repo_path?: string }): string {
  if ('repo_path' in repo && repo.repo_path) {
    return normalizeRepoPath(repo.repo_path)
  }

  if ('folder_path' in repo && repo.folder_path) {
    return `${repo.folder_path}/${repo.name}`
  }
  
  return repo.name
}
