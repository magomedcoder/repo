<script setup lang="ts">
import LoginPanel from '@/components/LoginPanel.vue'

defineProps<{
  loading?: boolean
  error?: string
}>()
</script>

<template>
  <p v-if="loading" class="py-6 text-sm text-ink-muted">{{ $t('common.loading') }}</p>
  <LoginPanel v-else-if="error === '__401__'" />
  <p v-else-if="error" class="py-4 text-sm text-warn">{{ error }}</p>
  <slot v-else />
</template>
