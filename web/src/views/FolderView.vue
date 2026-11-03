<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { foldersApi } from '@/api'
import type { Folder, FolderContents, RepoSummary } from '@/api/types'
import { useAuth } from '@/composables/useAuth'
import { folderKey } from '@/lib/folderNav'
import { idleAge } from '@/lib/idle'
import { localizeError } from '@/i18n'
import { repoLocation } from '@/lib/repoNav'
import { normalizeRepoPath, repositoryPath } from '@/lib/repoPath'

const props = defineProps<{
  folderPath: string
  initial?: FolderContents | null
}>()

const auth = useAuth()
const folder = ref<Folder | null>(null)
const repos = ref<RepoSummary[]>([])
const error = ref('')
const loading = ref(true)

const path = computed(() => normalizeRepoPath(props.folderPath))

const entries = computed(() => {
  const owner = auth.user.value?.username || ''
  const folderPath = folder.value ? folderKey(folder.value) : path.value
  return [...repos.value]
    .map((repo) => ({
      id: repo.id,
      name: repo.name,
      description: repo.description,
      repo_path: repo.repo_path
        ? normalizeRepoPath(repo.repo_path)
        : repositoryPath({ ...repo, folder_path: folderPath, owner }),
      idle: idleAge(repo.updated_at),
    }))
    .sort((a, b) => a.name.localeCompare(b.name))
})

function isLast(i: number) {
  return i === entries.value.length - 1
}

function applyContents(res: FolderContents) {
  folder.value = res.folder ?? null
  repos.value = res.repos ?? []
}

async function load() {
  loading.value = true
  error.value = ''
  if (!path.value) {
    error.value = localizeError(new Error('invalid'), 'errors.loadFolder')
    loading.value = false
    return
  }
  try {
    applyContents(await foldersApi.get(path.value))
  } catch (err) {
    error.value = localizeError(err, 'errors.loadFolder')
  } finally {
    loading.value = false
  }
}

let usedInitial = false

watch(
  () => props.folderPath,
  () => {
    if (props.initial && !usedInitial) {
      usedInitial = true
      applyContents(props.initial)
      loading.value = false
      error.value = ''
      return
    }
    load()
  },
  { immediate: true },
)
</script>

<template>
  <div>
    <p v-if="loading" class="text-sm text-ink-muted">{{ $t('common.loading') }}</p>
    <p v-else-if="error" class="text-sm text-warn">{{ error }}</p>
    <div v-else class="overflow-x-auto font-mono text-[90%] leading-[1.6]">
      <div>
        <RouterLink to="/" class="font-bold text-ink no-underline hover:underline">{{ $t('nav.root') }}/</RouterLink>
      </div>

      <div class="flex items-baseline gap-1">
        <span class="select-none text-ink-muted">└──</span>
        <span class="font-bold text-ink">{{ folder?.name || $t('folder.fallback') }}/</span>
      </div>

      <p v-if="!entries.length" class="pl-8 text-ink-muted">{{ $t('folder.empty') }}</p>

      <div
        v-for="(entry, i) in entries"
        :key="entry.id"
        class="flex min-w-0 items-baseline gap-3 hover:bg-row-hover"
      >
        <span class="shrink-0 select-none whitespace-pre text-ink-muted">
          {{ isLast(i) ? '    └──' : '    ├──' }}
        </span>
        <RouterLink
          :to="repoLocation('repo', entry.repo_path)"
          class="shrink-0 text-ink no-underline hover:underline"
        >
          {{ entry.name }}
        </RouterLink>
        <span class="min-w-0 flex-1 truncate text-ink-muted">{{ entry.description }}</span>
        <span class="shrink-0 whitespace-nowrap text-ink-muted">{{ entry.idle }}</span>
        <span class="flex shrink-0 gap-3 whitespace-nowrap text-ink-muted [&_a]:no-underline hover:[&_a]:underline">
          <RouterLink :to="repoLocation('repo', entry.repo_path)">{{ $t('index.summary') }}</RouterLink>
          <RouterLink :to="repoLocation('repo-commits', entry.repo_path)">{{ $t('index.log') }}</RouterLink>
        </span>
      </div>
    </div>
  </div>
</template>
