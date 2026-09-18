<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { localizeError } from '@/i18n'
import LocaleSwitch from '@/components/LocaleSwitch.vue'

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
    error.value = localizeError(err, 'errors.registerFailed')
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
          <p class="mt-2 text-ink-muted">{{ $t('auth.registerLead') }}</p>
        </div>
        <LocaleSwitch />
      </div>

      <form class="panel mt-8 space-y-4 p-6" @submit.prevent="submit">
        <h1 class="font-display text-2xl font-bold">{{ $t('auth.register') }}</h1>
        <label class="block text-sm">
          <span class="mb-1 block font-medium">{{ $t('auth.username') }}</span>
          <input
            v-model="username"
            class="input font-mono"
            required
            autocomplete="username"
          />
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-medium">{{ $t('auth.email') }}</span>
          <input
            v-model="email"
            type="email"
            class="input"
            required
            autocomplete="email"
          />
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-medium">{{ $t('auth.password') }}</span>
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
          {{ busy ? $t('auth.creating') : $t('auth.createAccount') }}
        </button>
        <p class="text-center text-sm text-ink-muted">
          {{ $t('auth.haveAccount') }}
          <RouterLink :to="{ name: 'login' }" class="font-semibold text-moss hover:underline">{{ $t('auth.loginTitle') }}</RouterLink>
        </p>
      </form>
    </div>
  </div>
</template>
