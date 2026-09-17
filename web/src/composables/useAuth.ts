import { computed, reactive } from 'vue'
import { ApiError } from '@/api/client'
import { authApi } from '@/api'
import type { User } from '@/api/types'

const state = reactive<{
  user: User | null
  ready: boolean
  loading: boolean
}>({
  user: null,
  ready: false,
  loading: false,
})

export function useAuth() {
  const user = computed(() => state.user)
  const isAuthenticated = computed(() => !!state.user)
  const ready = computed(() => state.ready)

  async function fetchMe() {
    state.loading = true
    try {
      state.user = await authApi.me()
    } catch (err) {
      if (err instanceof ApiError && (err.status === 401 || err.status === 403)) {
        state.user = null
      } else {
        state.user = null
      }
    } finally {
      state.loading = false
      state.ready = true
    }
  }

  async function login(login: string, password: string) {
    state.user = await authApi.login({ login, password })
    state.ready = true
  }

  async function register(username: string, email: string, password: string) {
    state.user = await authApi.register({ username, email, password })
    state.ready = true
  }

  async function logout() {
    try {
      await authApi.logout()
    } finally {
      state.user = null
    }
  }

  return {
    user,
    isAuthenticated,
    ready,
    loading: computed(() => state.loading),
    fetchMe,
    login,
    register,
    logout,
  }
}
