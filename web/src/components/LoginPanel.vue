<script setup lang="ts">
import { btnPrimary, field } from '@/lib/classes'
import { ref } from 'vue'
import { useAuth } from '@/composables/useAuth'
import { localizeError } from '@/i18n'

const auth = useAuth()
const login = ref('')
const password = ref('')
const error = ref('')
const busy = ref(false)

async function submit() {
  error.value = ''
  busy.value = true
  try {
    await auth.login(login.value.trim(), password.value)
    window.location.reload()
  } catch (err) {
    error.value = localizeError(err, 'errors.loginFailed')
    busy.value = false
  }
}
</script>

<template>
  <form class="max-w-md py-4" @submit.prevent="submit">
    <h1 class="mb-3">{{ $t('auth.loginTitle') }}</h1>
    <label class="mb-3 block text-sm">
      <span class="mb-1 block font-semibold">{{ $t('auth.usernameOrEmail') }}</span>
      <input v-model="login" required :class="field" autocomplete="username" />
    </label>
    <label class="mb-3 block text-sm">
      <span class="mb-1 block font-semibold">{{ $t('auth.password') }}</span>
      <input v-model="password" type="password" required :class="field" autocomplete="current-password" />
    </label>
    <p v-if="error" class="mb-2 text-sm text-warn">{{ error }}</p>
    <button :class="btnPrimary" type="submit" :disabled="busy">{{ $t('auth.signIn') }}</button>
  </form>
</template>
