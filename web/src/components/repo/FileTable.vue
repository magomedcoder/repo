<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import type { RouteLocationRaw } from 'vue-router'
import type { TreeEntry } from '@/api/types'
import { formatBytes } from '@/lib/bytes'
import { entryRoute } from '@/lib/repoNav'

const props = defineProps<{
  entries: TreeEntry[]
  repoPath: string
  refName: string
  parentTo?: RouteLocationRaw | null
  tree?: boolean
}>()

const sorted = computed(() =>
  [...props.entries].sort((a, b) => {
    if (a.type !== b.type) {
      return a.type === 'tree' ? -1 : 1
    }
    
    return a.name.localeCompare(b.name)
  }),
)

function isLast(i: number) {
  return i === sorted.value.length - 1
}

function label(entry: TreeEntry) {
  return entry.type === 'tree' ? `${entry.name}/` : entry.name
}
</script>

<template>
  <div v-if="tree" class="overflow-x-auto font-mono text-[90%] leading-[1.6]">
    <div class="mb-0.5 flex flex-wrap items-center justify-between gap-2">
      <span class="font-bold text-ink">{{ refName }}/</span>
      <slot name="actions" />
    </div>
    <div v-if="parentTo" class="pl-0">
      <RouterLink :to="parentTo" class="text-ink-muted no-underline hover:underline">..</RouterLink>
    </div>
    <p v-if="!sorted.length" class="pl-4 text-ink-muted">{{ $t('repo.empty') }}</p>
    <div
      v-for="(entry, i) in sorted"
      :key="entry.path"
      class="flex min-w-0 items-baseline gap-3 hover:bg-row-hover"
    >
      <span class="shrink-0 select-none text-ink-muted">{{ isLast(i) ? '└──' : '├──' }}</span>
      <RouterLink
        :to="entryRoute(repoPath, refName, entry)"
        class="min-w-0 truncate text-ink no-underline hover:underline"
        :class="entry.type === 'tree' && 'font-bold'"
      >
        {{ label(entry) }}
      </RouterLink>
      <span v-if="entry.type === 'blob'" class="ml-auto shrink-0 whitespace-nowrap text-ink-muted">
        {{ formatBytes(entry.size) }}
      </span>
    </div>
  </div>
  <div v-else class="overflow-x-auto">
    <table class="w-full border-collapse [&_th]:px-2 [&_th]:py-1 [&_th]:text-left [&_th]:align-baseline [&_th]:font-medium [&_th]:text-ink-muted [&_td]:border-0 [&_td]:px-2 [&_td]:py-1 [&_td]:align-baseline [&_tbody_tr:nth-child(odd)]:bg-paper [&_tbody_tr:nth-child(even)]:bg-row-alt [&_tbody_tr:hover]:bg-row-hover">
      <thead>
        <tr>
          <th>{{ $t('index.mode') }}</th>
          <th>{{ $t('index.name') }}</th>
          <th>{{ $t('index.size') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="parentTo">
          <td />
          <td>
            <RouterLink :to="parentTo">..</RouterLink>
          </td>
          <td />
        </tr>
        <tr v-for="entry in sorted" :key="entry.path">
          <td class="font-mono">{{ entry.mode }}</td>
          <td>
            <RouterLink :to="entryRoute(repoPath, refName, entry)">{{ label(entry) }}</RouterLink>
          </td>
          <td>{{ entry.type === 'blob' ? formatBytes(entry.size) : '' }}</td>
        </tr>
        <tr v-if="!sorted.length">
          <td colspan="3">{{ $t('repo.empty') }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
