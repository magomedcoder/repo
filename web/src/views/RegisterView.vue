<script setup lang="ts">
import { btnPrimary, field } from '@/lib/classes'
import { ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { localizeError } from '@/i18n'
import AppFooter from '@/components/AppFooter.vue'
import AppHeader from '@/components/AppHeader.vue'

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
  <div class="flex min-h-screen flex-col bg-white">
    <AppHeader />
    <div class="flex flex-1 justify-center px-3 py-8">
      <form class="w-full max-w-sm border border-line bg-white" @submit.prevent="submit">
        <h1 class="border-b border-line bg-paper-2 px-3 py-1.5 text-[13px] font-semibold">{{ $t('auth.register') }}</h1>
        <div class="p-3">
        <label class="mb-3 block text-[13px]">
          <span class="mb-1 block font-semibold">{{ $t('auth.username') }}</span>
          <input
            v-model="username"
            :class="field"
            required
            autocomplete="username"
          />
        </label>
        <label class="mb-3 block text-[13px]">
          <span class="mb-1 block font-semibold">{{ $t('auth.email') }}</span>
          <input
            v-model="email"
            type="email"
            :class="field"
            required
            autocomplete="email"
          />
        </label>
        <label class="mb-3 block text-[13px]">
          <span class="mb-1 block font-semibold">{{ $t('auth.password') }}</span>
          <input
            v-model="password"
            type="password"
            :class="field"
            required
            minlength="8"
            autocomplete="new-password"
          />
        </label>
        <p v-if="error" class="mb-3 text-[13px] text-warn">{{ error }}</p>
        <div>
          <button
            type="submit"
            :class="btnPrimary"
            :disabled="busy"
          >
            {{ busy ? $t('auth.creating') : $t('auth.createAccount') }}
          </button>
          <p class="mt-3">
            {{ $t('auth.haveAccount') }}
            <RouterLink :to="{ name: 'login' }">{{ $t('auth.signIn') }}</RouterLink>
          </p>
        </div>
        </div>
      </form>
    </div>
    <AppFooter />
  </div>
</template>
