<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRegistration } from '@/composables/useRegistration'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { usePageMeta } from '@/composables/usePageMeta'
import { localizeError } from '@/i18n'
import { safeRedirect } from '@/lib/safeRedirect'
import AppFooter from '@/components/layout/AppFooter.vue'
import AppHeader from '@/components/layout/AppHeader.vue'
import GuestAuthCard from '@/components/layout/GuestAuthCard.vue'
import Icon from '@/components/ui/Icon.vue'

const auth = useAuth()
const route = useRoute()
const router = useRouter()
const { allowRegistration, fetchAllowRegistration } = useRegistration()
usePageMeta(() => ({ 
  title: 'Sign in', 
  noindex: true, 
  canonicalPath: '/login' 
}))

onMounted(() => {
  void fetchAllowRegistration()
})

const login = ref('')
const password = ref('')
const error = ref('')
const busy = ref(false)

async function submit() {
  error.value = ''
  busy.value = true
  try {
    await auth.login(login.value.trim(), password.value)
    await router.replace(safeRedirect(route.query.redirect))
  } catch (err) {
    error.value = localizeError(err, 'errors.loginFailed')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="app-shell">
    <AppHeader />
    <div class="content-block flex flex-1 flex-col items-center justify-center px-1 py-8 sm:px-3 sm:py-10">
      <GuestAuthCard @submit="submit">
        <h1 class="text-center text-[1.35rem] font-bold leading-tight text-ink">
          {{ $t('auth.loginTitle') }}
        </h1>

        <label class="block text-sm">
          <span class="mb-1.5 block font-medium text-ink-muted">{{ $t('auth.usernameOrEmail') }}</span>
          <input
            v-model="login"
            class="field"
            required
            autocomplete="username"
          >
        </label>

        <label class="block text-sm">
          <span class="mb-1.5 block font-medium text-ink-muted">{{ $t('auth.password') }}</span>
          <input
            v-model="password"
            type="password"
            class="field"
            required
            autocomplete="current-password"
          >
        </label>

        <p v-if="error" class="text-sm text-warn">{{ error }}</p>

        <button type="submit" class="btn-primary w-full py-2" :disabled="busy">
          <Icon name="logout" size="sm" class="rotate-180" />
          {{ busy ? $t('auth.signingIn') : $t('auth.signIn') }}
        </button>

        <p v-if="allowRegistration" class="text-center text-sm text-ink-muted">
          {{ $t('auth.noAccount') }}
          <RouterLink
            :to="{
              name: 'register',
              query: route.query.redirect ? { redirect: route.query.redirect } : {},
            }"
          >
            {{ $t('auth.register') }}
          </RouterLink>
        </p>
      </GuestAuthCard>
    </div>
    <AppFooter />
  </div>
</template>
