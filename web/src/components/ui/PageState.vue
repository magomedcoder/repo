<script setup lang="ts">
import type { PageError } from '@/i18n'
import LoginPanel from '@/components/LoginPanel.vue'

defineProps<{
  loading?: boolean
  error?: PageError | string | null
}>()

function isUnauthorized(error: PageError | string | null | undefined) {
  return typeof error === 'object' && error?.kind === 'unauthorized'
}

function message(error: PageError | string | null | undefined) {
  if (!error || isUnauthorized(error)) {
    return ''
  }

  if (typeof error === 'string') {
    return error
  }
  
  if (error.kind === 'message') {
    return error.text
  }

  return ''
}
</script>

<template>
  <p v-if="loading" class="py-8 text-sm text-ink-muted">{{ $t('common.loading') }}</p>
  <LoginPanel v-else-if="isUnauthorized(error)" />
  <p v-else-if="message(error)" class="py-4 text-sm text-warn">{{ message(error) }}</p>
  <slot v-else />
</template>
