<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { usersApi } from '@/api'
import PageState from '@/components/ui/PageState.vue'
import { useAuth } from '@/composables/useAuth'
import { localizeError } from '@/i18n'
import Icon from '@/components/ui/Icon.vue'

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
  <div>
    <div class="page-card mb-4">
      <p class="mb-1 text-[85%] text-ink-muted">
        <RouterLink :to="{ name: 'profile' }" class="text-link hover:underline">{{ $t('profile.title') }}</RouterLink>
        <span> / </span>
      </p>
      <h1 class="inline-flex items-center gap-2 text-[140%] font-bold text-ink"><Icon name="mail" class="text-ink-muted" />{{ $t('profile.updateEmail') }}</h1>
    </div>

    <PageState :loading="loading" :error="error">
      <form class="page-card space-y-3" @submit.prevent="save">
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('profile.email') }}</span>
          <input
            v-model="email"
            type="email"
            required
            class="field"
          >
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('profile.currentPassword') }}</span>
          <input
            v-model="currentPassword"
            type="password"
            required
            class="field"
            autocomplete="current-password"
          >
        </label>
        <div class="flex flex-wrap items-center gap-3">
          <button
            type="submit"
            class="btn-primary"
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
