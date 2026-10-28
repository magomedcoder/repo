<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRegistration } from '@/composables/useRegistration'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { localizeError } from '@/i18n'
import { safeRedirect } from '@/lib/safeRedirect'
import AppFooter from '@/components/AppFooter.vue'
import AppHeader from '@/components/AppHeader.vue'

const auth = useAuth()
const route = useRoute()
const router = useRouter()
const { allowRegistration, fetchAllowRegistration } = useRegistration()

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
  <div class="flex min-h-0 flex-1 flex-col">
    <AppHeader />
    <div class="flex flex-1 justify-start py-6">
      <form class="w-full max-w-sm" @submit.prevent="submit">
        <h1 class="mb-3 text-[140%] font-bold text-ink">{{ $t('auth.loginTitle') }}</h1>
        <table class="w-full border-collapse">
          <tbody>
            <tr>
              <td class="whitespace-nowrap py-1 pr-3 align-middle">{{ $t('auth.usernameOrEmail') }}</td>
              <td class="w-full py-1">
                <input
                  v-model="login"
                  class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none"
                  required
                  autocomplete="username"
                />
              </td>
            </tr>
            <tr>
              <td class="whitespace-nowrap py-1 pr-3 align-middle">{{ $t('auth.password') }}</td>
              <td class="py-1">
                <input
                  v-model="password"
                  type="password"
                  class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none"
                  required
                  autocomplete="current-password"
                />
              </td>
            </tr>
            <tr>
              <td />
              <td class="py-2">
                <p v-if="error" class="mb-2 text-warn">{{ error }}</p>
                <button type="submit" class="inline-flex items-center justify-center border border-ink bg-ink px-2.5 py-1 text-[10pt] text-paper hover:bg-ink-muted disabled:cursor-not-allowed disabled:opacity-50" :disabled="busy">
                  {{ busy ? $t('auth.signingIn') : $t('auth.signIn') }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-if="allowRegistration" class="mt-3 text-sm">
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
      </form>
    </div>
    <AppFooter />
  </div>
</template>
