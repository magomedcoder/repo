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

export function repositoryPath(repo: Pick<Repository, 'name' | 'repo_path'> | RepoSummary): string {
  if (repo.repo_path) {
    return normalizeRepoPath(repo.repo_path)
  }

  return repo.name
}
