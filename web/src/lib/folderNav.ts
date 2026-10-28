import type { RouteLocationRaw } from 'vue-router'
import type { Folder } from '@/api/types'
import { encodeRepoPath, normalizeRepoPath } from '@/lib/repoPath'

export function folderKey(folder: Pick<Folder, 'path' | 'slug'> | string): string {
  if (typeof folder === 'string') {
    return normalizeRepoPath(folder)
  }
  
  return normalizeRepoPath(folder.path || folder.slug || '')
}

export function folderLocation(folder: Pick<Folder, 'path' | 'slug'> | string): RouteLocationRaw {
  const key = folderKey(folder)
  return key ? '/' + encodeRepoPath(key) : { name: 'folders' }
}
