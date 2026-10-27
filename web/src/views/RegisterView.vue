<script setup lang="ts">
import { btnPrimary, field } from '@/lib/classes'
import { onMounted, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { useRegistration } from '@/composables/useRegistration'
import { localizeError } from '@/i18n'
import { safeRedirect } from '@/lib/safeRedirect'
import AppFooter from '@/components/AppFooter.vue'
import AppHeader from '@/components/AppHeader.vue'

const auth = useAuth()
const route = useRoute()
const router = useRouter()
const { fetchAllowRegistration } = useRegistration()

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
  <div class="flex min-h-0 flex-1 flex-col">
    <AppHeader />
    <div v-if="checking" class="flex flex-1 py-6 text-ink-muted">{{ $t('common.loading') }}</div>
    <div v-else class="flex flex-1 justify-start py-6">
      <form class="w-full max-w-sm" @submit.prevent="submit">
        <h1 class="mb-3 text-[140%] font-bold text-ink">{{ $t('auth.registerTitle') }}</h1>
        <table class="w-full border-collapse">
          <tbody>
            <tr>
              <td class="whitespace-nowrap py-1 pr-3 align-middle">{{ $t('auth.username') }}</td>
              <td class="w-full py-1">
                <input
                  v-model="username"
                  :class="field"
                  required
                  autocomplete="username"
                />
              </td>
            </tr>
            <tr>
              <td class="whitespace-nowrap py-1 pr-3 align-middle">{{ $t('auth.email') }}</td>
              <td class="py-1">
                <input
                  v-model="email"
                  type="email"
                  :class="field"
                  required
                  autocomplete="email"
                />
              </td>
            </tr>
            <tr>
              <td class="whitespace-nowrap py-1 pr-3 align-middle">{{ $t('auth.password') }}</td>
              <td class="py-1">
                <input
                  v-model="password"
                  type="password"
                  :class="field"
                  required
                  autocomplete="new-password"
                />
              </td>
            </tr>
            <tr>
              <td />
              <td class="py-2">
                <p v-if="error" class="mb-2 text-warn">{{ error }}</p>
                <button type="submit" :class="btnPrimary" :disabled="busy">
                  {{ busy ? $t('auth.creating') : $t('auth.createAccount') }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
        <p class="mt-3 text-sm">
          {{ $t('auth.haveAccount') }}
          <RouterLink :to="{ name: 'login', query: route.query.redirect ? { redirect: route.query.redirect } : {} }">
            {{ $t('auth.signIn') }}
          </RouterLink>
        </p>
      </form>
    </div>
    <AppFooter />
  </div>
</template>
