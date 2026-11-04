import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import type { Repository } from '@/api/types'
import { useAuth } from '@/composables/useAuth'

export function useCanWrite(repo?: MaybeRefOrGetter<Repository | null | undefined>) {
  const auth = useAuth()

  const isAuthenticated = computed(() => auth.isAuthenticated.value)
  const username = computed(() => auth.user.value?.username || '')

  const current = computed(() => (repo ? toValue(repo) : null))

  const canWrite = computed(() => !!current.value?.can_write)
  const canAdmin = computed(() => !!current.value?.can_admin)

  return {
    user: auth.user,
    isAuthenticated,
    username,
    canWrite,
    canAdmin,
  }
}
