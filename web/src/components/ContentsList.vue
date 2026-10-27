<script setup lang="ts">
import { computed } from 'vue'
import { dataTable, textButton } from '@/lib/classes'
import { RouterLink } from 'vue-router'
import type { Folder, Repository, RepoSummary } from '@/api/types'
import { useAuth } from '@/composables/useAuth'
import { idleAge } from '@/lib/idle'
import { toContentRow } from '@/lib/contentRow'

const props = defineProps<{
  folders?: Folder[]
  repos: (RepoSummary | Repository)[]
  folderPath?: string
  repoOwner?: string
  showFolder?: boolean
}>()

const auth = useAuth()

const rows = computed(() =>
  props.repos.map((repo) => toContentRow(repo, props.folderPath, props.repoOwner ?? auth.user.value?.username)),
)

const emit = defineEmits<{
  sort: [key: string]
}>()

function folderTo(row: ReturnType<typeof toContentRow>) {
  if (row.folder_id == null) {
    return null
  }
  return { name: 'folder' as const, params: { id: String(row.folder_id) } }
}

function repoTo(row: ReturnType<typeof toContentRow>, page: 'repo' | 'repo-commits' | 'repo-tree') {
  return {
    name: page,
    params: {
      repoPath: row.repo_path,
    },
  }
}

function idleOf(row: ReturnType<typeof toContentRow>) {
  return idleAge(row.last_activity_at || row.updated_at)
}

const colSpan = props.showFolder ? 6 : 5
</script>

<template>
  <div>
    <table :class="dataTable">
      <thead>
        <tr>
          <th class="w-[18%]"><button type="button" :class="textButton" @click="emit('sort', 'name')">{{ $t('index.name') }}</button></th>
          <th><button type="button" :class="textButton" @click="emit('sort', 'description')">{{ $t('index.description') }}</button></th>
          <th v-if="showFolder" class="w-[12%]">{{ $t('index.folder') }}</th>
          <th class="w-[12%]">{{ $t('index.owner') }}</th>
          <th class="w-[12%]"><button type="button" :class="textButton" @click="emit('sort', 'idle')">{{ $t('index.idle') }}</button></th>
          <th class="w-[22%]">{{ $t('index.links') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="folder in folders" :key="`folder-${folder.id}`">
          <td>
            <RouterLink :to="{ name: 'folder', params: { id: String(folder.id) } }">{{ folder.name }}</RouterLink>
          </td>
          <td></td>
          <td v-if="showFolder"></td>
          <td></td>
          <td></td>
          <td></td>
        </tr>
        <tr v-for="row in rows" :key="`repo-${row.id}`">
          <td class="truncate">
            <RouterLink :to="repoTo(row, 'repo')">{{ row.name }}</RouterLink>
          </td>
          <td class="truncate" :title="row.description || undefined">{{ row.description }}</td>
          <td v-if="showFolder" class="truncate">
            <RouterLink v-if="folderTo(row)" :to="folderTo(row)!">{{ row.folder_path }}</RouterLink>
          </td>
          <td class="truncate">{{ row.owner }}</td>
          <td class="whitespace-nowrap">{{ idleOf(row) }}</td>
          <td class="whitespace-nowrap [&_a]:mr-[0.6em]">
            <RouterLink :to="repoTo(row, 'repo')">{{ $t('index.summary') }}</RouterLink>
            <RouterLink :to="repoTo(row, 'repo-commits')">{{ $t('index.log') }}</RouterLink>
            <RouterLink :to="repoTo(row, 'repo-tree')">{{ $t('index.tree') }}</RouterLink>
          </td>
        </tr>
        <tr v-if="!rows.length && !(folders?.length)">
          <td :colspan="colSpan">{{ $t('list.noRepos') }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
