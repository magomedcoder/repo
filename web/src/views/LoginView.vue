<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { localizeError } from '@/i18n'
import Icon from '@/components/ui/Icon.vue'
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
  <div class="flex min-h-screen flex-col bg-paper">
    <div class="flex justify-end px-4 py-3">
      <LocaleSwitch />
    </div>
    <div class="flex flex-1 items-start justify-center px-4 pb-16">
      <div class="w-full max-w-sm">
        <div class="mb-6 flex flex-col items-center gap-2 text-header">
          <span class="scale-150"><Icon name="mark" /></span>
          <p class="text-2xl font-semibold text-ink">Repo</p>
        </div>
        <form class="rounded-md border border-line bg-white p-4" @submit.prevent="submit">
          <h1 class="mb-4 text-center text-xl font-semibold">{{ $t('auth.loginTitle') }}</h1>
          <label class="mb-3 block text-sm">
            <span class="mb-1 block font-semibold">{{ $t('auth.usernameOrEmail') }}</span>
            <input
              v-model="login"
              class="input"
              required
              autocomplete="username"
            />
          </label>
          <label class="mb-3 block text-sm">
            <span class="mb-1 block font-semibold">{{ $t('auth.password') }}</span>
            <input
              v-model="password"
              type="password"
              class="input"
              required
              autocomplete="current-password"
            />
          </label>
          <p v-if="error" class="mb-3 text-sm text-warn">{{ error }}</p>
          <button type="submit" class="btn-primary w-full" :disabled="busy">
            {{ busy ? $t('auth.signingIn') : $t('auth.signIn') }}
          </button>
        </form>
        <p class="mt-4 rounded-md border border-line bg-white px-4 py-3 text-center text-sm">
          {{ $t('auth.noAccount') }}
          <RouterLink
            :to="{ name: 'register' }"
            class="text-accent hover:underline"
          >{{ $t('auth.createAccount') }}</RouterLink>
        </p>
      </div>
    </div>
  </div>
</template>
