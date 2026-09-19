import type { RouteLocationRaw } from 'vue-router'
import type { TreeEntry } from '@/api/types'

export function entryRoute(owner: string, repoPath: string, refName: string, entry: TreeEntry): RouteLocationRaw {
  if (entry.type === 'tree') {
    return {
      name: 'repo-tree',
      params: { owner, repoPath },
      query: {
        ref: refName,
        path: entry.path
      },
    }
  }

  return {
    name: 'repo-blob',
    params: { owner, repoPath },
    query: {
      ref: refName,
      path: entry.path
    },
  }
}

export function repoHome(owner: string, repoPath: string): RouteLocationRaw {
  return {
    name: 'repo',
    params: { owner, repoPath }
  }
}
