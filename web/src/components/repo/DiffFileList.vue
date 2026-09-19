<script setup lang="ts">
import type { FileDiff } from '@/api/types'
import { diffStatus } from '@/i18n'

defineProps<{
  files: FileDiff[]
}>()

function lineClass(line: string) {
  if (line.startsWith('+++') || line.startsWith('---')) {
    return 'text-ink-muted'
  }

  if (line.startsWith('@@')) {
    return 'bg-accent-soft text-accent'
  }

  if (line.startsWith('+')) {
    return 'bg-[#dafbe1]'
  }

  if (line.startsWith('-')) {
    return 'bg-[#ffebe9]'
  }

  return ''
}
</script>

<template>
  <article
    v-for="file in files"
    :key="file.path + file.status"
    class="overflow-hidden rounded-md border border-line bg-white"
  >
    <header class="flex flex-wrap items-center gap-2 border-b border-line bg-paper px-4 py-2 text-sm">
      <span class="rounded-full border border-line bg-white px-2 py-0.5 text-xs font-medium">{{ diffStatus(file.status) }}</span>
      <span class="font-mono">{{ file.path }}</span>
      <span v-if="file.old_path" class="font-mono text-ink-muted">{{ $t('commit.from', { path: file.old_path }) }}</span>
    </header>
    <pre class="overflow-x-auto p-0 font-mono text-xs leading-5"><div
      v-for="(line, i) in (file.patch || '').split('\n')"
      :key="i"
      class="px-4"
      :class="lineClass(line)"
    >{{ line || ' ' }}</div></pre>
  </article>
</template>
