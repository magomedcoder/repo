<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { localizeError } from '@/i18n'
import LocaleSwitch from '@/components/LocaleSwitch.vue'

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
    error.value = localizeError(err, 'errors.loginFailed')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center px-4 py-10">
    <div class="w-full max-w-md">
      <div class="flex items-start justify-between gap-4">
        <div>
          <p class="font-display text-4xl font-extrabold tracking-tight text-ink">Repo</p>
          <p class="mt-2 text-ink-muted">{{ $t('auth.loginLead') }}</p>
        </div>
        <LocaleSwitch />
      </div>

      <form class="panel mt-8 space-y-4 p-6" @submit.prevent="submit">
        <h1 class="font-display text-2xl font-bold">{{ $t('auth.loginTitle') }}</h1>
        <label class="block text-sm">
          <span class="mb-1 block font-medium">{{ $t('auth.usernameOrEmail') }}</span>
          <input
            v-model="login"
            class="input"
            required
            autocomplete="username"
          />
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-medium">{{ $t('auth.password') }}</span>
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
          {{ busy ? $t('auth.signingIn') : $t('auth.signIn') }}
        </button>
        <p class="text-center text-sm text-ink-muted">
          {{ $t('auth.noAccount') }}
          <RouterLink :to="{ name: 'register' }" class="font-semibold text-moss hover:underline">{{ $t('auth.register') }}</RouterLink>
        </p>
      </form>
    </div>
  </div>
</template>
