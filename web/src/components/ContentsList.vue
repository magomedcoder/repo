<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import type { Repository, RepoSummary } from '@/api/types'
import { useAuth } from '@/composables/useAuth'
import { idleAge } from '@/lib/idle'
import { toContentRow } from '@/lib/contentRow'
import { folderLocation } from '@/lib/folderNav'
import { repoLocation } from '@/lib/repoNav'

const props = defineProps<{
  repos: (RepoSummary | Repository)[]
  showFolder?: boolean
}>()

const auth = useAuth()

const rows = computed(() => props.repos.map((repo) => toContentRow(repo, undefined, auth.user.value?.username)))

const emit = defineEmits<{
  sort: [key: string]
}>()

function repoTo(row: ReturnType<typeof toContentRow>, page: 'repo' | 'repo-commits' | 'repo-tree') {
  return repoLocation(page, row.repo_path)
}

function idleOf(row: ReturnType<typeof toContentRow>) {
  return idleAge(row.last_activity_at || row.updated_at)
}

function canOpenFolder(row: ReturnType<typeof toContentRow>) {
  return !!auth.user.value?.username && row.owner === auth.user.value.username
}
</script>

<template>
  <div>
    <table class="w-full border-collapse [&_th]:px-2 [&_th]:py-1 [&_th]:text-left [&_th]:align-baseline [&_th]:font-medium [&_th]:text-ink-muted [&_td]:border-0 [&_td]:px-2 [&_td]:py-1 [&_td]:align-baseline [&_tbody_tr:nth-child(odd)]:bg-paper [&_tbody_tr:nth-child(even)]:bg-row-alt [&_tbody_tr:hover]:bg-row-hover">
      <thead>
        <tr>
          <th class="w-[28%]">
            <button type="button" class="cursor-pointer border-0 bg-transparent p-0 text-[10pt] font-medium text-ink hover:underline" @click="emit('sort', 'name')">{{ $t('index.name') }}</button>
          </th>
          <th><button type="button" class="cursor-pointer border-0 bg-transparent p-0 text-[10pt] font-medium text-ink hover:underline" @click="emit('sort', 'description')">{{ $t('index.description') }}</button></th>
          <th class="w-[12%]">{{ $t('index.owner') }}</th>
          <th class="w-[12%]"><button type="button" class="cursor-pointer border-0 bg-transparent p-0 text-[10pt] font-medium text-ink hover:underline" @click="emit('sort', 'idle')">{{ $t('index.idle') }}</button></th>
          <th class="w-[22%]">{{ $t('index.links') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="row in rows" :key="`repo-${row.id}`">
          <td class="truncate" :title="showFolder && row.folder_path ? `${row.folder_path}/${row.name}` : row.name">
            <template v-if="showFolder && row.folder_path">
              <RouterLink v-if="canOpenFolder(row)" :to="folderLocation(row.folder_path)">{{ row.folder_path }}</RouterLink>
              <span v-else>{{ row.folder_path }}</span>
              <span>/</span>
            </template>
            <RouterLink :to="repoTo(row, 'repo')">{{ row.name }}</RouterLink>
          </td>
          <td class="truncate" :title="row.description || undefined">{{ row.description }}</td>
          <td class="truncate">{{ row.owner }}</td>
          <td class="whitespace-nowrap">{{ idleOf(row) }}</td>
          <td class="whitespace-nowrap [&_a]:mr-[0.6em]">
            <RouterLink :to="repoTo(row, 'repo')">{{ $t('index.summary') }}</RouterLink>
            <RouterLink :to="repoTo(row, 'repo-commits')">{{ $t('index.log') }}</RouterLink>
            <RouterLink :to="repoTo(row, 'repo-tree')">{{ $t('index.tree') }}</RouterLink>
          </td>
        </tr>
        <tr v-if="!rows.length">
          <td colspan="5">{{ $t('list.noRepos') }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
