<script setup lang="ts">
import { computed } from 'vue'
import type { BlobContent, BlameResult } from '@/api/types'

const props = defineProps<{
  blob: BlobContent
  showBlame: boolean
  blame: BlameResult | null
  blameLoading: boolean
  blameError: string
  filename: string
}>()

const textLines = computed(() => (props.blob.content || '').split('\n'))

function lineClass(line: string, filename: string) {
  const ext = filename.split('.').pop()?.toLowerCase() || ''
  const comment = ext === 'py' || ext === 'sh' ? /^\s*#/ : /^\s*(\/\/|\/\*|\*)/
  if (comment.test(line)) {
    return 'text-ink-muted'
  }

  if (/^\s*(func|class|def|import|package|type|const|let|var|return)\b/.test(line)) {
    return 'text-accent'
  }

  return ''
}

function shortSha(sha: string) {
  return sha.slice(0, 7)
}
</script>

<template>
  <div v-if="showBlame && !blob.is_binary">
    <p v-if="blameLoading" class="px-2 py-2 text-[9pt] text-ink-muted">{{ $t('common.loading') }}</p>
    <p v-else-if="blameError" class="px-2 py-2 text-[9pt] text-warn">{{ blameError }}</p>
    <div v-else-if="blame" class="max-h-[75vh] overflow-auto">
      <table class="w-full border-collapse font-mono text-[11px] leading-4">
        <tbody>
          <tr
            v-for="line in blame.lines"
            :key="line.line"
            class="hover:bg-row-hover"
          >
            <td class="w-8 select-none whitespace-nowrap border-r border-line/70 bg-path px-1.5 py-0 text-right tabular-nums text-ink-muted">
              {{ line.line }}
            </td>
            <td class="whitespace-nowrap px-1.5 py-0 text-ink-muted">{{ shortSha(line.commit_sha) }}</td>
            <td class="max-w-[8rem] truncate whitespace-nowrap px-1.5 py-0 text-ink-muted">{{ line.author_name }}</td>
            <td class="whitespace-pre px-1.5 py-0 text-ink">{{ line.content || ' ' }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>

  <div v-else-if="!blob.is_binary" class="max-h-[75vh] overflow-auto">
    <table class="w-full border-collapse font-mono text-[11px] leading-4">
      <tbody>
        <tr v-for="(line, i) in textLines" :key="i" class="hover:bg-row-hover">
          <td class="w-8 select-none whitespace-nowrap border-r border-line/70 bg-path px-1.5 py-0 text-right tabular-nums text-ink-muted">
            {{ i + 1 }}
          </td>
          <td class="whitespace-pre px-2 py-0" :class="lineClass(line, filename)">{{ line || ' ' }}</td>
        </tr>
      </tbody>
    </table>
  </div>

  <p v-else class="px-2 py-2 text-[9pt] text-ink-muted">{{ $t('blob.binary') }}</p>
</template>
