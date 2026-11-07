<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { localizeError } from '@/i18n'
import GuestAuthCard from '@/components/layout/GuestAuthCard.vue'
import Icon from '@/components/ui/Icon.vue'

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
    await router.replace({ 
      name: 'home' 
    })
  } catch (err) {
    error.value = localizeError(err, 'errors.loginFailed')
    busy.value = false
  }
}
</script>

<template>
  <GuestAuthCard @submit="submit">
    <h1 class="text-center text-[1.35rem] font-bold leading-tight text-ink">
      {{ $t('auth.loginTitle') }}
    </h1>

    <label class="block text-sm">
      <span class="mb-1.5 block font-medium text-ink-muted">{{ $t('auth.usernameOrEmail') }}</span>
      <input v-model="login" required class="field" autocomplete="username">
    </label>

    <label class="block text-sm">
      <span class="mb-1.5 block font-medium text-ink-muted">{{ $t('auth.password') }}</span>
      <input v-model="password" type="password" required class="field" autocomplete="current-password">
    </label>

    <p v-if="error" class="text-sm text-warn">{{ error }}</p>

    <button class="btn-primary w-full py-2" type="submit" :disabled="busy">
      <Icon name="logout" size="sm" class="rotate-180" />
      {{ busy ? $t('auth.signingIn') : $t('auth.signIn') }}
    </button>
  </GuestAuthCard>
</template>
