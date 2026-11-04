<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import { authApi } from '@/api'
import { localizeError } from '@/i18n'

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
  <div class="mx-auto max-w-xl">
    <div class="mb-4 border-b border-line pb-3">
      <p class="mb-1 text-[85%] text-ink-muted">
        <RouterLink :to="{ name: 'profile' }" class="text-link hover:underline">{{ $t('profile.title') }}</RouterLink>
        <span> / </span>
      </p>
      <h1 class="text-[140%] font-bold text-ink">{{ $t('profile.changePasswordTitle') }}</h1>
    </div>

    <form class="space-y-3 border border-line p-3 sm:p-4" @submit.prevent="save">
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
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('profile.newPassword') }}</span>
        <input
          v-model="nextPassword"
          type="password"
          required
          minlength="8"
          class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none"
          autocomplete="new-password"
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
      <p v-if="error" class="text-sm text-warn">{{ error }}</p>
      <p v-else-if="saved" class="text-sm text-ink-muted">{{ $t('common.ok') }}</p>
    </form>
  </div>
</template>
