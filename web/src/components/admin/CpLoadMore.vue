<script setup lang="ts">
defineProps<{
  loading?: boolean
  hasMore?: boolean
}>()

defineEmits<{
  load: []
}>()
</script>

<template>
  <p v-if="hasMore" class="mt-3">
    <button type="button" class="btn" :disabled="loading" @click="$emit('load')">
      {{ loading ? $t('common.loading') : $t('common.loadMore') }}
    </button>
  </p>
</template>
