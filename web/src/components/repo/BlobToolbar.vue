<script setup lang="ts">
import { RouterLink } from 'vue-router'
import type { BlobContent } from '@/api/types'
import PathCrumbs from '@/components/repo/PathCrumbs.vue'
import Icon from '@/components/ui/Icon.vue'
import { formatBytes } from '@/lib/bytes'
import { repoLocation } from '@/lib/repoNav'

defineProps<{
  repoPath: string
  refName: string
  filePath: string
  blob: BlobContent
  showBlame: boolean
  parentDir: string
  rawHref: string
  rawFilename: string
}>()

const emit = defineEmits<{
  toggleBlame: []
}>()
</script>

<template>
  <div class="surface-header flex-wrap gap-x-2 gap-y-1 px-2 py-1">
    <PathCrumbs
      class="min-w-0 flex-1"
      :repo-path="repoPath"
      :ref-name="refName"
      :path="filePath"
    />
    <div class="flex shrink-0 items-center gap-1.5 text-[9pt] text-ink-muted">
      <span class="tabular-nums">{{ formatBytes(blob.size) }}</span>

      <span>{{ blob.is_binary ? $t('common.binary') : blob.encoding }}</span>
    </div>
    <div class="flex shrink-0 gap-1">
      <button
        v-if="!blob.is_binary"
        type="button"
        :class="[
          'btn px-1.5 py-0.5 text-[9pt]',
          showBlame && 'bg-paper-2',
        ]"
        @click="emit('toggleBlame')"
      >
        <Icon name="eye" size="sm" />
        {{ showBlame ? $t('blob.hideBlame') : $t('blob.blame') }}
      </button>
      <RouterLink
        class="btn px-1.5 py-0.5 text-[9pt] no-underline hover:no-underline"
        :to="repoLocation('repo', repoPath, { query: { ref: refName, path: parentDir || undefined } })"
      >
        <Icon name="folder" size="sm" />
        {{ $t('blob.tree') }}
      </RouterLink>
      <a
        class="btn px-1.5 py-0.5 text-[9pt] no-underline hover:no-underline"
        :href="rawHref"
        :download="rawFilename"
        target="_blank"
        rel="noreferrer"
      >
        <Icon name="download" size="sm" />
        {{ $t('blob.raw') }}
      </a>
    </div>
  </div>
</template>
