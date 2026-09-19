<script setup lang="ts">
import { RouterLink } from 'vue-router'
import type { CommitInfo } from '@/api/types'
import { formatDate } from '@/i18n'

defineProps<{
  commits: CommitInfo[]
  owner: string
  repoPath: string
}>()

function shortSha(sha: string) {
  return sha.slice(0, 7)
}

function firstLine(message: string) {
  return message.split('\n')[0]
}
</script>

<template>
  <ul class="divide-y divide-line overflow-hidden rounded-md border border-line bg-white">
    <li
      v-for="commit in commits"
      :key="commit.sha"
      class="flex items-start gap-3 px-4 py-3 hover:bg-paper"
    >
      <span class="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-accent-soft text-[11px] font-semibold text-accent">
        {{ commit.author_name.slice(0, 1).toUpperCase() }}
      </span>
      <div class="min-w-0 flex-1">
        <RouterLink
          :to="{
            name: 'repo-commit',
            params: {
              owner, repoPath,
              sha: commit.sha
            }
          }"
          class="font-semibold text-ink hover:text-accent hover:underline"
        >
          {{ firstLine(commit.message) }}
        </RouterLink>
        <p class="mt-0.5 text-xs text-ink-muted">{{ commit.author_name }} {{ $t('commits.authored') }} {{ formatDate(commit.authored_at) }}</p>
      </div>
      <RouterLink
        :to="{
          name: 'repo-commit',
          params: { owner, repoPath, sha: commit.sha }
        }"
        class="shrink-0 rounded-md border border-line bg-paper px-2 py-0.5 font-mono text-xs text-accent hover:underline"
      >
        {{ shortSha(commit.sha) }}
      </RouterLink>
    </li>
    <li v-if="!commits.length" class="px-4 py-8 text-center text-sm text-ink-muted">{{ $t('commits.empty') }}</li>
  </ul>
</template>
