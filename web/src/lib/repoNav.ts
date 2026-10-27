import type { RouteLocationRaw } from 'vue-router'
import type { TreeEntry } from '@/api/types'

export function entryRoute(repoPath: string, refName: string, entry: TreeEntry): RouteLocationRaw {
  if (entry.type === 'tree') {
    return {
      name: 'repo-tree',
      params: { repoPath },
      query: {
        ref: refName,
        path: entry.path
      },
    }
  }

  return {
    name: 'repo-blob',
    params: { repoPath },
    query: {
      ref: refName,
      path: entry.path
    },
  }
}
