<script setup lang="ts">
defineProps<{
  isPrivate: boolean
}>()
</script>

<template>
  <span class="inline-flex items-center rounded-full border border-line px-2 py-0.5 text-xs font-medium capitalize text-ink-muted">
    {{ isPrivate ? $t('common.private') : $t('common.public') }}
  </span>
</template>
