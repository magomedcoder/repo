<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  languages: Record<string, number>
}>()

const palette = [
  '#3178c6',
  '#f1e05a',
  '#3572a5',
  '#00add8',
  '#dea584',
  '#b07219',
  '#e34c26',
  '#563d7c',
  '#89e051',
  '#c6538c',
]

const rows = computed(() => {
  const entries = Object.entries(props.languages)
  const total = entries.reduce((sum, [, bytes]) => sum + bytes, 0) || 1
  return entries
    .map(([name, bytes], i) => ({
      name,
      pct: (bytes / total) * 100,
      color: palette[i % palette.length]!,
    }))
    .sort((a, b) => b.pct - a.pct)
})
</script>

<template>
  <div class="space-y-2">
    <div class="flex h-2 overflow-hidden rounded-full bg-white/40">
      <span
        v-for="row in rows"
        :key="row.name"
        class="h-full"
        :style="{ width: `${Math.max(row.pct, 0.6)}%`, background: row.color }"
        :title="`${row.name} ${row.pct.toFixed(1)}%`"
      />
    </div>
    <ul class="flex flex-wrap gap-x-3 gap-y-1 text-[12px]">
      <li
        v-for="row in rows"
        :key="row.name"
        class="inline-flex items-center gap-1.5"
      >
        <span
          class="inline-block h-2.5 w-2.5 rounded-full"
          :style="{ background: row.color }"
        />
        <span class="font-medium text-ink">{{ row.name }}</span>
        <span class="tabular-nums text-ink-muted">{{ row.pct.toFixed(1) }}%</span>
      </li>
    </ul>
  </div>
</template>
