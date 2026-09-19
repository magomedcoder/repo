<script setup lang="ts">
import { RouterLink } from 'vue-router'
import type { RouteLocationRaw } from 'vue-router'
import type { TreeEntry } from '@/api/types'
import Icon from '@/components/ui/Icon.vue'
import { entryRoute } from '@/lib/repoNav'

defineProps<{
  entries: TreeEntry[]
  owner: string
  repoPath: string
  refName: string
  parentTo?: RouteLocationRaw | null
}>()
</script>

<template>
  <div class="overflow-hidden rounded-md border border-line bg-white">
    <div class="flex flex-wrap items-center justify-between gap-2 border-b border-line bg-paper px-3 py-2">
      <div class="flex min-w-0 items-center gap-2">
        <span class="inline-flex items-center gap-1 rounded-md border border-line bg-white px-2 py-1 font-mono text-xs font-semibold">
          {{ refName }}
        </span>
        <slot name="meta" />
      </div>
      <slot name="actions" />
    </div>
    <ul class="divide-y divide-line">
      <li v-if="parentTo">
        <RouterLink
          :to="parentTo"
          class="flex items-center gap-2 px-4 py-2 hover:bg-paper"
        >
          <Icon name="folder" class="text-accent" />
          <span class="font-medium text-accent">..</span>
        </RouterLink>
      </li>
      <li v-for="entry in entries" :key="entry.path">
        <RouterLink
          :to="entryRoute(owner, repoPath, refName, entry)"
          class="flex items-center gap-2 px-4 py-2 hover:bg-paper"
        >
          <Icon
            :name="entry.type === 'tree' ? 'folder' : 'file'"
            :class="entry.type === 'tree' ? 'text-accent' : 'text-ink-muted'"
          />
          <span class="min-w-0 flex-1 truncate font-medium text-accent">{{ entry.name }}</span>
          <span
            v-if="entry.type === 'blob'"
            class="font-mono text-xs text-ink-muted"
          >{{ $t('common.sizeB', { n: entry.size }) }}</span>
        </RouterLink>
      </li>
      <li v-if="!entries.length" class="px-4 py-8 text-center text-sm text-ink-muted">{{ $t('repo.empty') }}</li>
    </ul>
  </div>
</template>
