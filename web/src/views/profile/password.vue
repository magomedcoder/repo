<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import { authApi } from '@/api'
import { localizeError } from '@/i18n'
import Icon from '@/components/ui/Icon.vue'

const currentPassword = ref('')
const nextPassword = ref('')
const error = ref('')
const saved = ref(false)
const saving = ref(false)

async function save() {
  error.value = ''
  saved.value = false
  saving.value = true
  try {
    await authApi.password({
      current: currentPassword.value,
      next: nextPassword.value,
    })
    currentPassword.value = ''
    nextPassword.value = ''
    saved.value = true
  } catch (err) {
    error.value = localizeError(err, 'errors.saveFailed')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div>
    <div class="page-card mb-4">
      <p class="mb-1 text-[85%] text-ink-muted">
        <RouterLink :to="{ name: 'profile' }" class="text-link hover:underline">{{ $t('profile.title') }}</RouterLink>
        <span> / </span>
      </p>
      <h1 class="inline-flex items-center gap-2 text-[140%] font-bold text-ink"><Icon name="lock" class="text-ink-muted" />{{ $t('profile.changePasswordTitle') }}</h1>
    </div>

    <form class="page-card space-y-3" @submit.prevent="save">
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
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('profile.newPassword') }}</span>
        <input
          v-model="nextPassword"
          type="password"
          required
          minlength="8"
          class="field"
          autocomplete="new-password"
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
      <p v-if="error" class="text-sm text-warn">{{ error }}</p>
      <p v-else-if="saved" class="text-sm text-ink-muted">{{ $t('common.ok') }}</p>
    </form>
  </div>
</template>
