import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import type { Repository } from '@/api/types'
import { useAuth } from '@/composables/useAuth'

export function useCanWrite(repo?: MaybeRefOrGetter<Repository | null | undefined>) {
  const auth = useAuth()

  const isAuthenticated = computed(() => auth.isAuthenticated.value)
  const username = computed(() => auth.user.value?.username || '')

  const isOwner = computed(() => {
    const current = repo ? toValue(repo) : null
    return !!username.value && !!current?.owner && username.value === current.owner
  })

  return {
    user: auth.user,
    isAuthenticated,
    username,
    isOwner,
  }
}
