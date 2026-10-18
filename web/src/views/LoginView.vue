<script setup lang="ts">
import { btnPrimary, field } from '@/lib/classes'
import { ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { localizeError } from '@/i18n'
import AppFooter from '@/components/AppFooter.vue'
import AppHeader from '@/components/AppHeader.vue'

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
  <div class="flex min-h-screen flex-col bg-white">
    <AppHeader />
    <div class="flex flex-1 justify-center px-3 py-8">
      <form class="w-full max-w-sm border border-line bg-white" @submit.prevent="submit">
        <h1 class="border-b border-line bg-paper-2 px-3 py-1.5 text-[13px] font-semibold">{{ $t('auth.loginTitle') }}</h1>
        <div class="p-3">
        <label class="mb-3 block text-[13px]">
          <span class="mb-1 block font-semibold">{{ $t('auth.usernameOrEmail') }}</span>
          <input
            v-model="login"
            :class="field"
            required
            autocomplete="username"
          />
        </label>
        <label class="mb-3 block text-[13px]">
          <span class="mb-1 block font-semibold">{{ $t('auth.password') }}</span>
          <input
            v-model="password"
            type="password"
            :class="field"
            required
            autocomplete="current-password"
          />
        </label>
        <p v-if="error" class="mb-3 text-[13px] text-warn">{{ error }}</p>
        <div>
          <button type="submit" :class="btnPrimary" :disabled="busy">
            {{ busy ? $t('auth.signingIn') : $t('auth.signIn') }}
          </button>
          <p class="mt-3">
            {{ $t('auth.noAccount') }}
            <RouterLink :to="{ name: 'register' }">{{ $t('auth.createAccount') }}</RouterLink>
          </p>
        </div>
        </div>
      </form>
    </div>
    <AppFooter />
  </div>
</template>
