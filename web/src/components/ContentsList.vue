<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import type { Repository, RepoSummary } from '@/api/types'
import Icon from '@/components/ui/Icon.vue'
import type { IconName } from '@/components/ui/Icon.vue'
import GlassTable from '@/components/ui/GlassTable.vue'
import { useAuth } from '@/composables/useAuth'
import { idleAge } from '@/lib/idle'
import { toContentRow } from '@/lib/contentRow'
import { folderLocation } from '@/lib/folderNav'
import { repoLocation } from '@/lib/repoNav'

const props = defineProps<{
  repos: (RepoSummary | Repository)[]
  showFolder?: boolean
  sort?: string
  emptyText?: string
  title?: string
  hint?: string
}>()

const auth = useAuth()

const rows = computed(() => props.repos.map((repo) => toContentRow(repo, undefined, auth.user.value?.username)))

const privacy = computed(() => {
  const map = new Map<number, boolean>()
  for (const repo of props.repos) {
    if ('is_private' in repo) {
      map.set(repo.id, !!repo.is_private)
    }
  }
  return map
})

const emit = defineEmits<{
  sort: [key: string]
}>()

function repoTo(row: ReturnType<typeof toContentRow>, page: 'repo' | 'repo-commits') {
  return repoLocation(page, row.repo_path)
}

function idleOf(row: ReturnType<typeof toContentRow>) {
  return idleAge(row.last_activity_at || row.updated_at)
}

function canOpenFolder(row: ReturnType<typeof toContentRow>) {
  return !!auth.user.value?.username && row.owner === auth.user.value.username
}

function sortClass(key: string) {
  return props.sort === key
    ? 'inline-flex items-center gap-1 font-bold text-ink'
    : 'inline-flex items-center gap-1 font-medium text-ink-muted hover:text-ink'
}

function sortIcon(key: string): IconName {
  return props.sort === key ? 'chevron-down' : 'sort'
}

function isPrivate(id: number) {
  return privacy.value.get(id) === true
}
</script>

<template>
  <div class="glass overflow-hidden">
    <div v-if="title" class="border-b border-line-strong/40 px-3 py-2 sm:px-4">
      <h1 class="font-bold text-ink">{{ title }}</h1>
      <p v-if="hint" class="text-[85%] text-ink-muted">{{ hint }}</p>
    </div>
    <div class="min-w-0 overflow-x-auto">
    <GlassTable :glass="false" min-width-class="min-w-[28rem]">
      <thead>
        <tr>
          <th class="w-[30%]">
            <button
              type="button"
              class="cursor-pointer border-0 bg-transparent p-0 text-[10pt]"
              :class="sortClass('name')"
              :aria-pressed="sort === 'name'"
              @click="emit('sort', 'name')"
            >
              {{ $t('index.name') }}
              <Icon :name="sortIcon('name')" size="xs" :class="sort === 'name' ? 'text-ink' : 'opacity-50'" />
            </button>
          </th>
          <th>
            <button
              type="button"
              class="cursor-pointer border-0 bg-transparent p-0 text-[10pt]"
              :class="sortClass('description')"
              :aria-pressed="sort === 'description'"
              @click="emit('sort', 'description')"
            >
              {{ $t('index.description') }}
              <Icon :name="sortIcon('description')" size="xs" :class="sort === 'description' ? 'text-ink' : 'opacity-50'" />
            </button>
          </th>
          <th class="w-[14%]">
            <button
              type="button"
              class="cursor-pointer border-0 bg-transparent p-0 text-[10pt]"
              :class="sortClass('idle')"
              :aria-pressed="sort === 'idle'"
              @click="emit('sort', 'idle')"
            >
              {{ $t('index.idle') }}
              <Icon :name="sortIcon('idle')" size="xs" :class="sort === 'idle' ? 'text-ink' : 'opacity-50'" />
            </button>
          </th>
          <th class="w-[20%]">{{ $t('index.links') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="row in rows" :key="`repo-${row.id}`">
          <td class="truncate" :title="showFolder && row.folder_path ? `${row.folder_path}/${row.name}` : row.name">
            <template v-if="showFolder && row.folder_path">
              <RouterLink v-if="canOpenFolder(row)" :to="folderLocation(row.folder_path)">{{ row.folder_path }}</RouterLink>
              <span v-else class="text-ink-muted">{{ row.folder_path }}</span>
              <span class="text-ink-muted">/</span>
            </template>
            <RouterLink :to="repoTo(row, 'repo')">{{ row.name }}</RouterLink>
            <span
              v-if="isPrivate(row.id)"
              class="ml-1 inline-flex align-middle text-ink-muted"
              :title="$t('home.private')"
            >
              <Icon name="lock" size="sm" />
            </span>
          </td>
          <td class="truncate" :title="row.description || undefined">{{ row.description }}</td>
          <td class="whitespace-nowrap">{{ idleOf(row) }}</td>
          <td class="whitespace-nowrap [&_a]:mr-[0.6em]">
            <RouterLink :to="repoTo(row, 'repo')">{{ $t('index.summary') }}</RouterLink>
            <RouterLink :to="repoTo(row, 'repo-commits')">{{ $t('index.log') }}</RouterLink>
          </td>
        </tr>
        <tr v-if="!rows.length">
          <td colspan="4">{{ emptyText || $t('list.noRepos') }}</td>
        </tr>
      </tbody>
    </GlassTable>
    </div>
  </div>
</template>
