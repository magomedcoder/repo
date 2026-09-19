<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { localizeError } from '@/i18n'
import Icon from '@/components/ui/Icon.vue'
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
    await router.replace({
      name: 'home'
    })
  } catch (err) {
    error.value = localizeError(err, 'errors.registerFailed')
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
          <h1 class="mb-4 text-center text-xl font-semibold">{{ $t('auth.register') }}</h1>
          <label class="mb-3 block text-sm">
            <span class="mb-1 block font-semibold">{{ $t('auth.username') }}</span>
            <input
              v-model="username"
              class="input font-mono"
              required
              autocomplete="username"
            />
          </label>
          <label class="mb-3 block text-sm">
            <span class="mb-1 block font-semibold">{{ $t('auth.email') }}</span>
            <input
              v-model="email"
              type="email"
              class="input"
              required
              autocomplete="email"
            />
          </label>
          <label class="mb-3 block text-sm">
            <span class="mb-1 block font-semibold">{{ $t('auth.password') }}</span>
            <input
              v-model="password"
              type="password"
              class="input"
              required
              minlength="8"
              autocomplete="new-password"
            />
          </label>
          <p v-if="error" class="mb-3 text-sm text-warn">{{ error }}</p>
          <button
            type="submit"
            class="btn-primary w-full"
            :disabled="busy"
          >
            {{ busy ? $t('auth.creating') : $t('auth.createAccount') }}
          </button>
        </form>
        <p class="mt-4 rounded-md border border-line bg-white px-4 py-3 text-center text-sm">
          {{ $t('auth.haveAccount') }}
          <RouterLink :to="{ name: 'login' }" class="text-accent hover:underline">{{ $t('auth.signIn') }}</RouterLink>
        </p>
      </div>
    </div>
  </div>
</template>
