import type { RouteLocationRaw } from 'vue-router'
import type { TreeEntry } from '@/api/types'
import { encodeRepoPath } from '@/lib/repoPath'

export type RepoRouteKind =
  | 'repo'
  | 'repo-commits'
  | 'repo-commit'
  | 'repo-blob'
  | 'repo-refs'
  | 'repo-issues'
  | 'repo-issue'
  | 'repo-pulls'
  | 'repo-pull'
  | 'repo-settings'

export function repoLocation(
  kind: RepoRouteKind,
  repoPath: string,
  opts?: {
    number?: string | number
    sha?: string
    query?: Record<string, string | undefined>
  },
): RouteLocationRaw {
  const base = '/' + encodeRepoPath(repoPath)
  const suffixes: Record<RepoRouteKind, string> = {
    repo: '',
    'repo-commits': '/commits',
    'repo-commit': `/commits/${encodeURIComponent(opts?.sha || '')}`,
    'repo-blob': '/blob',
    'repo-refs': '/refs',
    'repo-issues': '/issues',
    'repo-issue': `/issues/${opts?.number ?? ''}`,
    'repo-pulls': '/pulls',
    'repo-pull': `/pulls/${opts?.number ?? ''}`,
    'repo-settings': '/settings',
  }
  const path = base + suffixes[kind]

  if (!opts?.query) {
    return path
  }

  const query: Record<string, string> = {}
  for (const [k, v] of Object.entries(opts.query)) {
    if (v) {
      query[k] = v
    }
  }

  return Object.keys(query).length ? { path, query } : path
}

export function entryRoute(repoPath: string, refName: string, entry: TreeEntry): RouteLocationRaw {
  return repoLocation(entry.type === 'tree' ? 'repo' : 'repo-blob', repoPath, {
    query: {
      ref: refName,
      path: entry.path,
    },
  })
}
