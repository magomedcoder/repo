<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { ApiError } from '@/api/client'

const auth = useAuth()
const router = useRouter()

const username = ref('')
const email = ref('')
const password = ref('')
const error = ref('')
const busy = ref(false)

async function submit() {
  error.value = ''
  busy.value = true
  try {
    await auth.register(username.value.trim(), email.value.trim(), password.value)
    await router.replace({ name: 'home' })
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'Registration failed'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center px-4 py-10">
    <div class="w-full max-w-md">
      <p class="font-display text-4xl font-extrabold tracking-tight text-ink">Repo</p>
      <p class="mt-2 text-ink-muted">Create an account and nest projects like folders.</p>

      <form class="panel mt-8 space-y-4 p-6" @submit.prevent="submit">
        <h1 class="font-display text-2xl font-bold">Register</h1>
        <label class="block text-sm">
          <span class="mb-1 block font-medium">Username</span>
          <input
            v-model="username"
            class="input font-mono"
            required
            autocomplete="username"
          />
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-medium">Email</span>
          <input
            v-model="email"
            type="email"
            class="input"
            required
            autocomplete="email"
          />
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-medium">Password</span>
          <input
            v-model="password"
            type="password"
            class="input"
            required
            minlength="8"
            autocomplete="new-password"
          />
        </label>
        <p v-if="error" class="text-sm text-warn">{{ error }}</p>
        <button
          type="submit"
          class="btn-primary w-full"
          :disabled="busy"
        >
          {{ busy ? 'Creating...' : 'Create account' }}
        </button>
        <p class="text-center text-sm text-ink-muted">
          Already have an account?
          <RouterLink :to="{ name: 'login' }" class="font-semibold text-moss hover:underline">Log in</RouterLink>
        </p>
      </form>
    </div>
  </div>
</template>
