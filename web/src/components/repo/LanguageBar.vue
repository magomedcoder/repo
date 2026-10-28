<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  languages: Record<string, number>
}>()

const rows = computed(() => {
  const entries = Object.entries(props.languages)
  const total = entries.reduce((sum, [, bytes]) => sum + bytes, 0) || 1
  return entries
    .map(([name, bytes]) => ({
      name,
      pct: (bytes / total) * 100,
    }))
    .sort((a, b) => b.pct - a.pct)
})
</script>

<template>
  <span>
    <template v-for="(row, i) in rows" :key="row.name">
      <span v-if="i">, </span>{{ row.name }}
      <span class="text-ink-muted"> {{ row.pct.toFixed(0) }}%</span>
    </template>
  </span>
</template>
