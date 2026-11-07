import { computed, reactive } from 'vue'
import { authApi } from '@/api'
import { clearCSRFToken } from '@/api/client'
import { RoleAdmin, type User } from '@/api/types'

const state = reactive<{
  user: User | null
  ready: boolean
}>({
  user: null,
  ready: false,
})

export function useAuth() {
  const user = computed(() => state.user)
  const isAuthenticated = computed(() => !!state.user)
  const isAdmin = computed(() => state.user?.role === RoleAdmin)
  const ready = computed(() => state.ready)

  async function fetchMe() {
    try {
      state.user = await authApi.me()
    } catch {
      state.user = null
    } finally {
      state.ready = true
    }
  }

  async function login(login: string, password: string) {
    clearCSRFToken()
    state.user = await authApi.login({ login, password })
    clearCSRFToken()
    state.ready = true
  }

  async function register(body: { username: string; email: string; password: string }) {
    clearCSRFToken()
    state.user = await authApi.register(body)
    clearCSRFToken()
    state.ready = true
    return state.user
  }

  async function logout() {
    try {
      await authApi.logout()
    } finally {
      clearCSRFToken()
      state.user = null
    }
  }

  function setUser(next: User | null) {
    state.user = next
  }

  return {
    user,
    isAuthenticated,
    isAdmin,
    ready,
    fetchMe,
    login,
    register,
    logout,
    setUser,
  }
}
