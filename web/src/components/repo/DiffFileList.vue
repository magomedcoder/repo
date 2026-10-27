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
    return 'bg-paper-2 text-ink'
  }

  if (line.startsWith('+')) {
    return 'bg-diff-add'
  }

  if (line.startsWith('-')) {
    return 'bg-diff-del'
  }

  return ''
}
</script>

<template>
  <div
    v-for="file in files"
    :key="file.path + file.status"
    class="mb-3 border-b border-line pb-3"
  >
    <div class="mb-1 bg-path px-2 py-1 font-mono text-[90%]">
      <span class="mr-2 font-sans text-ink-muted">{{ diffStatus(file.status) }}</span>
      {{ file.path }}
      <span v-if="file.old_path" class="text-ink-muted"> {{ $t('commit.from', { path: file.old_path }) }}</span>
    </div>
    <pre
      class="max-h-[32rem] overflow-auto p-0 font-mono text-[90%] leading-[1.35]"
    ><div
      v-for="(line, i) in (file.patch || '').split('\n').slice(0, 2000)"
      :key="i"
      class="px-2"
      :class="lineClass(line)"
    >{{ line || ' ' }}</div></pre>
  </div>
</template>
