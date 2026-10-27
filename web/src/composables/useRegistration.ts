import { ref } from 'vue'
import { authApi } from '@/api'

const allowRegistration = ref<boolean | null>(null)
let pending: Promise<boolean> | null = null

export function useRegistration() {
  async function fetchAllowRegistration(): Promise<boolean> {
    if (allowRegistration.value != null) {
      return allowRegistration.value
    }
    if (!pending) {
      pending = authApi.registration()
        .then((res) => {
          allowRegistration.value = !!res.allow_registration
          return allowRegistration.value
        })
        .catch(() => {
          allowRegistration.value = false
          return false
        })
        .finally(() => {
          pending = null
        })
    }
    return pending
  }

  return {
    allowRegistration,
    fetchAllowRegistration,
  }
}
