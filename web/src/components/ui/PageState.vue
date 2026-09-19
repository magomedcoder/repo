<script setup lang="ts">
defineProps<{
  loading?: boolean
  error?: string
}>()
</script>

<template>
  <p v-if="loading" class="py-6 text-sm text-ink-muted">{{ $t('common.loading') }}</p>
  <p v-else-if="error" class="py-4 text-sm text-warn">{{ error }}</p>
  <slot v-else />
</template>
