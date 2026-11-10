<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { usePageMeta } from '@/composables/usePageMeta'
import { useRegistration } from '@/composables/useRegistration'
import { localizeError } from '@/i18n'
import { safeRedirect } from '@/lib/safeRedirect'
import AppFooter from '@/components/layout/AppFooter.vue'
import AppHeader from '@/components/layout/AppHeader.vue'
import GuestAuthCard from '@/components/layout/GuestAuthCard.vue'
import Icon from '@/components/ui/Icon.vue'

const auth = useAuth()
const route = useRoute()
const router = useRouter()
const { fetchAllowRegistration } = useRegistration()
usePageMeta(() => ({ 
  title: 'Register', 
  noindex: true, 
  canonicalPath: '/register' 
}))

const username = ref('')
const email = ref('')
const password = ref('')
const error = ref('')
const busy = ref(false)
const checking = ref(true)

onMounted(async () => {
  const allowed = await fetchAllowRegistration()
  checking.value = false
  if (!allowed) {
    await router.replace({ name: 'login' })
  }
})

async function submit() {
  error.value = ''
  busy.value = true
  try {
    const user = await auth.register({
      username: username.value.trim(),
      email: email.value.trim(),
      password: password.value,
    })
    if (user) {
      await router.replace(safeRedirect(route.query.redirect))
    }
  } catch (err) {
    error.value = localizeError(err, 'errors.registerFailed')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="app-shell">
    <AppHeader />
    <div v-if="checking" class="content-block flex flex-1 items-center justify-center py-6 text-ink-muted">{{ $t('common.loading') }}</div>
    <div v-else class="content-block flex flex-1 flex-col items-center justify-center px-1 py-8 sm:px-3 sm:py-10">
      <GuestAuthCard @submit="submit">
        <h1 class="text-center text-[1.35rem] font-bold leading-tight text-ink">
          {{ $t('auth.registerTitle') }}
        </h1>

        <label class="block text-sm">
          <span class="mb-1.5 block font-medium text-ink-muted">{{ $t('auth.username') }}</span>
          <input
            v-model="username"
            class="field"
            required
            autocomplete="username"
          >
        </label>

        <label class="block text-sm">
          <span class="mb-1.5 block font-medium text-ink-muted">{{ $t('auth.email') }}</span>
          <input
            v-model="email"
            type="email"
            class="field"
            required
            autocomplete="email"
          >
        </label>

        <label class="block text-sm">
          <span class="mb-1.5 block font-medium text-ink-muted">{{ $t('auth.password') }}</span>
          <input
            v-model="password"
            type="password"
            class="field"
            required
            autocomplete="new-password"
          >
        </label>

        <p v-if="error" class="text-sm text-warn">{{ error }}</p>

        <button type="submit" class="btn-primary w-full py-2" :disabled="busy">
          <Icon name="plus" size="sm" />
          {{ busy ? $t('auth.creating') : $t('auth.createAccount') }}
        </button>

        <p class="text-center text-sm text-ink-muted">
          {{ $t('auth.haveAccount') }}
          <RouterLink :to="{ name: 'login', query: route.query.redirect ? { redirect: route.query.redirect } : {} }">
            {{ $t('auth.signIn') }}
          </RouterLink>
        </p>
      </GuestAuthCard>
    </div>
    <AppFooter />
  </div>
</template>
