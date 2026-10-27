import type { Ref } from 'vue'
import { inject } from 'vue'
import type { Repository } from '@/api/types'

function normalizeRepoPath(path: string): string {
  return path.replace(/^\/+|\/+$/g, '')
}

export function repositoryPath(r: Repository): string {
  if (r.repo_path) {
    return r.repo_path.replace(/\.git$/, '')
  }
  
  if (r.folder_path) {
    return `${r.folder_path}/${r.name}`
  }

  return `${r.owner}/${r.name}`
}

export function repoPathMatches(r: Repository | null, repoPath: string): boolean {
  if (!r) {
    return false
  }

  return normalizeRepoPath(repositoryPath(r)) === normalizeRepoPath(repoPath)
}

export function useInjectedRepo(): Ref<Repository | null> {
  const repo = inject<Ref<Repository | null>>('repo')
  if (!repo) {
    throw new Error('useInjectedRepo must be used within RepoShell')
  }

  return repo
}
