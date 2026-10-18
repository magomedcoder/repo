import { computed, ref } from 'vue'
import { useAuth } from '@/composables/useAuth'

const remembered = ref('')

export function rememberOwner(username: string) {
  if (username) {
    remembered.value = username
  }
}

export function useRepoOwner() {
  const auth = useAuth()
  return computed(() => auth.user.value?.username || remembered.value)
}
