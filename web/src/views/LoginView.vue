<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { ApiError } from '@/api/client'

const auth = useAuth()
const route = useRoute()
const router = useRouter()

const login = ref('')
const password = ref('')
const error = ref('')
const busy = ref(false)

async function submit() {
  error.value = ''
  busy.value = true
  try {
    await auth.login(login.value.trim(), password.value)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    await router.replace(redirect)
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'Login failed'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center px-4 py-10">
    <div class="w-full max-w-md">
      <p class="font-display text-4xl font-extrabold tracking-tight text-ink">Repo</p>
      <p class="mt-2 text-ink-muted">Sign in to your filesystem of repositories.</p>

      <form class="panel mt-8 space-y-4 p-6" @submit.prevent="submit">
        <h1 class="font-display text-2xl font-bold">Log in</h1>
        <label class="block text-sm">
          <span class="mb-1 block font-medium">Username or email</span>
          <input
            v-model="login"
            class="input"
            required
            autocomplete="username"
          />
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-medium">Password</span>
          <input
            v-model="password"
            type="password"
            class="input"
            required
            autocomplete="current-password"
          />
        </label>
        <p v-if="error" class="text-sm text-warn">{{ error }}</p>
        <button type="submit" class="btn-primary w-full" :disabled="busy">
          {{ busy ? 'Signing in...' : 'Sign in' }}
        </button>
        <p class="text-center text-sm text-ink-muted">
          No account?
          <RouterLink :to="{ name: 'register' }" class="font-semibold text-moss hover:underline">Register</RouterLink>
        </p>
      </form>
    </div>
  </div>
</template>
