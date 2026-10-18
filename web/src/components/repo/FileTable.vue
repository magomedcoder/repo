<script setup lang="ts">
import { dataTable } from '@/lib/classes'
import { RouterLink } from 'vue-router'
import type { RouteLocationRaw } from 'vue-router'
import type { TreeEntry } from '@/api/types'
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
  <div class="overflow-x-auto">
    <div class="mb-1 flex flex-wrap items-center justify-between gap-2 px-1">
      <div class="flex min-w-0 items-center gap-2">
        <span class="font-mono">{{ refName }}</span>
        <slot name="meta" />
      </div>
      <slot name="actions" />
    </div>
    <table :class="dataTable">
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
        <tr v-for="entry in entries" :key="entry.path">
          <td class="font-mono">{{ entry.mode }}</td>
          <td>
            <RouterLink :to="entryRoute(owner, repoPath, refName, entry)">{{ entry.name }}</RouterLink>
          </td>
          <td>{{ entry.type === 'blob' ? entry.size : '' }}</td>
        </tr>
        <tr v-if="!entries.length">
          <td colspan="3">{{ $t('repo.empty') }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
