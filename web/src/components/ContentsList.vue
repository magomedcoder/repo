<script setup lang="ts">
import { RouterLink } from 'vue-router'
import type { Folder, Repository, RepoSummary } from '@/api/types'
import { useAuth } from '@/composables/useAuth'
import Icon from '@/components/ui/Icon.vue'
import VisibilityBadge from '@/components/ui/VisibilityBadge.vue'

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
</script>

<template>
  <div class="space-y-6">
    <section v-if="folders.length" class="overflow-hidden rounded-md border border-line bg-white">
      <h2 class="border-b border-line bg-paper px-4 py-2 text-sm font-semibold">{{ $t('nav.folders') }}</h2>
      <ul class="divide-y divide-line">
        <li v-for="folder in folders" :key="folder.id">
          <RouterLink
            :to="{
              name: 'folder',
              params: {
                id: String(folder.id)
              }
            }"
            class="flex items-center gap-2 px-4 py-2.5 hover:bg-paper"
          >
            <Icon name="folder" class="text-accent" />
            <span class="font-semibold text-accent">{{ folder.name }}</span>
            <span class="truncate font-mono text-xs text-ink-muted">{{ folder.path }}</span>
          </RouterLink>
        </li>
      </ul>
    </section>

    <section class="overflow-hidden rounded-md border border-line bg-white">
      <h2 class="border-b border-line bg-paper px-4 py-2 text-sm font-semibold">{{ $t('nav.repositories') }}</h2>
      <ul v-if="repos.length" class="divide-y divide-line">
        <li v-for="repo in repos" :key="repo.id">
          <RouterLink
            :to="{
              name: 'repo',
              params: {
                owner: auth.user.value?.username ?? '',
                repoPath: repoPath(repo)
                }
            }"
            class="flex items-start gap-2 px-4 py-3 hover:bg-paper"
          >
            <Icon name="repo" class="mt-0.5 text-ink-muted" />
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-2">
                <span class="font-semibold text-accent">{{ repo.name }}</span>
                <VisibilityBadge :is-private="repo.is_private" />
              </div>
              <p v-if="repo.description" class="mt-0.5 text-xs text-ink-muted">{{ repo.description }}</p>
            </div>
          </RouterLink>
        </li>
      </ul>
      <p v-else class="px-4 py-8 text-center text-sm text-ink-muted">{{ $t('list.noRepos') }}</p>
    </section>
  </div>
</template>
