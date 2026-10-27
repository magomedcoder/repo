<script setup lang="ts">
import { btnPrimary, field } from '@/lib/classes'
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { localizeError } from '@/i18n'

const auth = useAuth()
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
    await router.replace({ name: 'home' })
  } catch (err) {
    error.value = localizeError(err, 'errors.loginFailed')
    busy.value = false
  }
}
</script>

<template>
  <form class="max-w-sm" @submit.prevent="submit">
    <h1 class="mb-3 text-[140%] font-bold text-ink">{{ $t('auth.loginTitle') }}</h1>
    <table class="w-full border-collapse">
      <tbody>
        <tr>
          <td class="whitespace-nowrap py-1 pr-3">{{ $t('auth.usernameOrEmail') }}</td>
          <td class="w-full py-1">
            <input v-model="login" required :class="field" autocomplete="username" />
          </td>
        </tr>
        <tr>
          <td class="whitespace-nowrap py-1 pr-3">{{ $t('auth.password') }}</td>
          <td class="py-1">
            <input v-model="password" type="password" required :class="field" autocomplete="current-password" />
          </td>
        </tr>
        <tr>
          <td />
          <td class="py-2">
            <p v-if="error" class="mb-2 text-warn">{{ error }}</p>
            <button :class="btnPrimary" type="submit" :disabled="busy">
              {{ busy ? $t('auth.signingIn') : $t('auth.signIn') }}
            </button>
          </td>
        </tr>
      </tbody>
    </table>
  </form>
</template>
