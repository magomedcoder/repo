<script setup lang="ts">
import { dataTable, textButton } from '@/lib/classes'
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import type { Folder, Repository, RepoSummary } from '@/api/types'
import { useAuth } from '@/composables/useAuth'
import { idleAge } from '@/lib/idle'

const props = defineProps<{
  folders: Folder[]
  repos: (RepoSummary | Repository)[]
  folderPath?: string
  repoOwner?: string
  groupByPath?: boolean
}>()

const auth = useAuth()

const emit = defineEmits<{
  sort: [key: string]
}>()

function logicalPath(repo: RepoSummary | Repository) {
  if ('repo_path' in repo && repo.repo_path) {
    return repo.repo_path.replace(/\.git$/, '')
  }
  return repoPath(repo)
}

function sectionOf(repo: RepoSummary | Repository) {
  const logical = logicalPath(repo)
  const slash = logical.lastIndexOf('/')
  return slash < 0 ? '' : logical.slice(0, slash)
}

function repoPath(repo: RepoSummary | Repository) {
  if ('repo_path' in repo && repo.repo_path) {
    return repo.repo_path.replace(/\.git$/, '')
  }
  if ('folder_path' in repo && repo.folder_path) {
    return `${repo.folder_path}/${repo.name}`
  }

  if (props.folderPath) {
    return `${props.folderPath}/${repo.name}`
  }

  return repo.name
}

function ownerForRepo(repo: RepoSummary | Repository) {
  if ('owner' in repo && repo.owner) {
    return repo.owner
  }

  return props.repoOwner || auth.user.value?.username || ''
}

function repoTo(repo: RepoSummary | Repository, page: 'repo' | 'repo-commits' | 'repo-tree') {
  return {
    name: page,
    params: {
      repoPath: repoPath(repo),
    },
  }
}

const rows = computed(() => {
  const out: Array<{ kind: 'section' | 'repo'; key: string; section?: string; repo?: RepoSummary | Repository }> = []
  let prev = '\0'
  const list = props.groupByPath
    ? [...props.repos].sort((a, b) => logicalPath(a).localeCompare(logicalPath(b)))
    : props.repos
  for (const repo of list) {
    const section = sectionOf(repo)
    if (props.groupByPath && section !== prev) {
      if (section) {
        out.push({ kind: 'section', key: `section-${section}`, section })
      }
      prev = section
    }
    out.push({ kind: 'repo', key: `repo-${repo.id}`, repo })
  }
  return out
})

function idleOf(repo: RepoSummary | Repository) {
  if ('last_activity_at' in repo) {
    return idleAge(repo.last_activity_at || repo.updated_at)
  }

  return idleAge(repo.updated_at)
}
</script>

<template>
  <div class="overflow-x-auto">
    <table :class="dataTable">
      <thead>
        <tr>
          <th><button type="button" :class="textButton" @click="emit('sort', 'name')">{{ $t('index.name') }}</button></th>
          <th><button type="button" :class="textButton" @click="emit('sort', 'description')">{{ $t('index.description') }}</button></th>
          <th>{{ $t('index.owner') }}</th>
          <th><button type="button" :class="textButton" @click="emit('sort', 'idle')">{{ $t('index.idle') }}</button></th>
          <th>{{ $t('index.links') }}</th>
        </tr>
      </thead>
      <tbody>
        <template v-for="row in rows" :key="row.key">
        <tr v-if="row.kind === 'section'" data-section>
          <td colspan="5">{{ row.section }}</td>
        </tr>
        <tr v-else-if="row.repo">
          <td>
            <RouterLink :to="repoTo(row.repo, 'repo')">{{ row.repo.name }}</RouterLink>
          </td>
          <td>{{ row.repo.description }}</td>
          <td>{{ ownerForRepo(row.repo) }}</td>
          <td class="whitespace-nowrap">{{ idleOf(row.repo) }}</td>
          <td class="whitespace-nowrap [&_a]:mr-[0.6em]">
            <RouterLink :to="repoTo(row.repo, 'repo')">{{ $t('index.summary') }}</RouterLink>
            <RouterLink :to="repoTo(row.repo, 'repo-commits')">{{ $t('index.log') }}</RouterLink>
            <RouterLink :to="repoTo(row.repo, 'repo-tree')">{{ $t('index.tree') }}</RouterLink>
          </td>
        </tr>
        </template>
        <tr v-for="folder in folders" :key="`folder-${folder.id}`">
          <td>
            <RouterLink :to="{ name: 'folder', params: { id: String(folder.id) } }">{{ folder.name }}</RouterLink>
          </td>
          <td></td>
          <td></td>
          <td></td>
          <td></td>
        </tr>
        <tr v-if="!repos.length && !folders.length">
          <td colspan="5">{{ $t('list.noRepos') }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
