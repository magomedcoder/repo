<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { usersApi } from '@/api'
import PageState from '@/components/ui/PageState.vue'
import { useAuth } from '@/composables/useAuth'
import { localizeError } from '@/i18n'

const auth = useAuth()
const email = ref('')
const currentPassword = ref('')
const error = ref('')
const formError = ref('')
const saved = ref(false)
const loading = ref(true)
const saving = ref(false)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const username = auth.user.value?.username
    if (!username) {
      throw new Error('unauthorized')
    }
    const profile = await usersApi.get(username)
    email.value = profile.email || ''
  } catch (err) {
    error.value = localizeError(err, 'errors.loadProfile')
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  formError.value = ''
  saved.value = false
  try {
    const profile = await usersApi.updateMe({
      email: email.value,
      current_password: currentPassword.value,
    })
    if (auth.user.value) {
      auth.setUser({
        ...auth.user.value,
        email: profile.email || email.value,
        has_avatar: profile.has_avatar,
      })
    }
    currentPassword.value = ''
    saved.value = true
  } catch (err) {
    formError.value = localizeError(err, 'errors.saveProfile')
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="mx-auto max-w-xl">
    <div class="mb-4 border-b border-line pb-3">
      <p class="mb-1 text-[85%] text-ink-muted">
        <RouterLink :to="{ name: 'profile' }" class="text-link hover:underline">{{ $t('profile.title') }}</RouterLink>
        <span> / </span>
      </p>
      <h1 class="text-[140%] font-bold text-ink">{{ $t('profile.updateEmail') }}</h1>
    </div>

    <PageState :loading="loading" :error="error">
      <form class="space-y-3 border border-line p-3 sm:p-4" @submit.prevent="save">
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('profile.email') }}</span>
          <input
            v-model="email"
            type="email"
            required
            class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none"
          >
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('profile.currentPassword') }}</span>
          <input
            v-model="currentPassword"
            type="password"
            required
            class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none"
            autocomplete="current-password"
          >
        </label>
        <div class="flex flex-wrap items-center gap-3">
          <button
            type="submit"
            class="inline-flex items-center justify-center border border-ink bg-ink px-2.5 py-1 text-[10pt] text-paper hover:bg-ink-muted disabled:cursor-not-allowed disabled:opacity-50"
            :disabled="saving"
          >
            {{ saving ? $t('profile.saving') : $t('profile.save') }}
          </button>
          <RouterLink :to="{ name: 'profile' }" class="text-[90%] text-ink-muted hover:underline">
            {{ $t('common.cancel') }}
          </RouterLink>
        </div>
        <p v-if="formError" class="text-sm text-warn">{{ formError }}</p>
        <p v-else-if="saved" class="text-sm text-ink-muted">{{ $t('common.ok') }}</p>
      </form>
    </PageState>
  </div>
</template>
