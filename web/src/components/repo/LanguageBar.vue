<script setup lang="ts">
defineProps<{
  languages: Record<string, number>
}>()

const palette = ['#1f883d', '#0969da', '#8250df', '#bf8700', '#cf222e', '#54aeff']

function rows(languages: Record<string, number>) {
  const entries = Object.entries(languages)
  const total = entries.reduce((sum, [, bytes]) => sum + bytes, 0) || 1
  return entries.map(([name, bytes], i) => ({
    name,
    bytes,
    pct: (bytes / total) * 100,
    color: palette[i % palette.length],
  })).sort((a, b) => b.bytes - a.bytes)
}
</script>

<template>
  <div>
    <div class="mb-2 flex h-2 overflow-hidden rounded-full">
      <span
        v-for="row in rows(languages)"
        :key="row.name"
        class="h-full"
        :style="{ width: `${row.pct}%`, background: row.color }"
      />
    </div>
    <ul class="space-y-1 text-xs">
      <li
        v-for="row in rows(languages)"
        :key="row.name"
        class="flex items-center justify-between gap-2"
      >
        <span class="inline-flex items-center gap-1.5">
          <span class="h-2 w-2 rounded-full" :style="{ background: row.color }" />
          {{ row.name }}
        </span>
        <span class="text-ink-muted">{{ row.pct.toFixed(1) }}%</span>
      </li>
    </ul>
  </div>
</template>
