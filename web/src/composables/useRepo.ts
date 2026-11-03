import { ref, toValue, watch, type MaybeRefOrGetter, type Ref } from 'vue'
import { reposApi } from '@/api'
import type { Repository } from '@/api/types'
import type { PageError } from '@/i18n'
import { pageError } from '@/i18n'
import { normalizeRepoPath, repositoryPath } from '@/lib/repoPath'

type CacheEntry = { repo: Repository; at: number }
const cache = new Map<string, CacheEntry>()
const CACHE_TTL_MS = 5_000

function cacheKey(path: string) {
  return normalizeRepoPath(path)
}

export function invalidateRepo(path: string) {
  cache.delete(cacheKey(path))
}

export function useRepo(repoPath: MaybeRefOrGetter<string>): {
  repo: Ref<Repository | null>
  loading: Ref<boolean>
  error: Ref<PageError | string>
  reload: () => Promise<void>
} {
  const repo = ref<Repository | null>(null)
  const loading = ref(true)
  const error = ref<PageError | string>('')

  async function reload(force = false) {
    const path = toValue(repoPath)
    const key = cacheKey(path)
    loading.value = true
    error.value = ''

    if (!force) {
      const hit = cache.get(key)
      if (hit && Date.now() - hit.at < CACHE_TTL_MS) {
        repo.value = hit.repo
        loading.value = false
        return
      }
    }

    repo.value = null
    try {
      const next = await reposApi.get(path)
      if (normalizeRepoPath(repositoryPath(next)) !== key) {
        return
      }

      cache.set(key, { 
        repo: next, 
        at: Date.now() 
      })
      repo.value = next
    } catch (err) {
      cache.delete(key)
      error.value = pageError(err, 'errors.loadRepository')
    } finally {
      loading.value = false
    }
  }

  watch(() => toValue(repoPath), () => reload(false), { immediate: true })

  return {
    repo,
    loading,
    error,
    reload: () => reload(true),
  }
}
