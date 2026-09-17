<script setup lang="ts">
import { RouterLink } from 'vue-router'
import type { Folder, Repository, RepoSummary } from '@/api/types'
import { useAuth } from '@/composables/useAuth'

const props = defineProps<{
  folders: Folder[]
  repos: (RepoSummary | Repository)[]
  folderPath?: string
}>()

const auth = useAuth()

function repoPath(repo: RepoSummary | Repository) {
  if ('folder_path' in repo && repo.folder_path) {
    return `${repo.folder_path}/${repo.name}`
  }

  if (props.folderPath) {
    return `${props.folderPath}/${repo.name}`
  }

  return repo.name
}

function repoLink(repo: RepoSummary | Repository) {
  return {
    name: 'repo' as const,
    params: {
      owner: auth.user.value?.username ?? '',
      repoPath: repoPath(repo),
    },
  }
}
</script>

<template>
  <div class="space-y-6">
    <section v-if="folders.length">
      <h2 class="mb-2 text-xs font-semibold uppercase tracking-wide text-ink-muted">Folders</h2>
      <ul class="divide-y divide-line overflow-hidden rounded-lg border border-line bg-white/80">
        <li v-for="folder in folders" :key="folder.id">
          <RouterLink
            :to="{
              name: 'folder',
              params: {
                id: String(folder.id)
              }
            }"
            class="flex items-center gap-3 px-4 py-3 transition hover:bg-moss-soft/40"
          >
            <span class="flex h-8 w-8 items-center justify-center rounded-md bg-paper-2 font-mono text-xs text-moss">dir</span>
            <div class="min-w-0">
              <p class="font-semibold">{{ folder.name }}</p>
              <p class="truncate font-mono text-xs text-ink-muted">{{ folder.path }}</p>
            </div>
          </RouterLink>
        </li>
      </ul>
    </section>

    <section>
      <h2 class="mb-2 text-xs font-semibold uppercase tracking-wide text-ink-muted">Repositories</h2>
      <ul v-if="repos.length" class="divide-y divide-line overflow-hidden rounded-lg border border-line bg-white/80">
        <li v-for="repo in repos" :key="repo.id">
          <RouterLink
            :to="repoLink(repo)"
            class="flex items-start gap-3 px-4 py-3 transition hover:bg-moss-soft/40"
          >
            <span class="mt-0.5 flex h-8 w-8 items-center justify-center rounded-md bg-paper-2 font-mono text-xs text-moss">git</span>
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-2">
                <p class="font-semibold">{{ repo.name }}</p>
                <span
                  class="rounded border px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide"
                  :class="repo.is_private ? 'border-warn/30 bg-warn-soft text-warn' : 'border-line text-ink-muted'"
                >
                  {{ repo.is_private ? 'private' : 'public' }}
                </span>
              </div>
              <p v-if="repo.description" class="mt-0.5 text-sm text-ink-muted">{{ repo.description }}</p>
            </div>
          </RouterLink>
        </li>
      </ul>
      <p v-else class="rounded-lg border border-dashed border-line px-4 py-8 text-center text-sm text-ink-muted">
        No repositories here yet.
      </p>
    </section>
  </div>
</template>
