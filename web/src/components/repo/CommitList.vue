<script setup lang="ts">
import { RouterLink } from 'vue-router'
import type { CommitInfo } from '@/api/types'
import { idleAge } from '@/lib/idle'
import { repoLocation } from '@/lib/repoNav'

defineProps<{
  commits: CommitInfo[]
  repoPath: string
}>()

function shortSha(sha: string) {
  return sha.slice(0, 12)
}

function firstLine(message: string) {
  return message.split('\n')[0]
}
</script>

<template>
  <div class="overflow-x-auto">
    <table class="w-full border-collapse [&_th]:px-2 [&_th]:py-1 [&_th]:text-left [&_th]:align-baseline [&_th]:font-medium [&_th]:text-ink-muted [&_td]:border-0 [&_td]:px-2 [&_td]:py-1 [&_td]:align-baseline [&_tbody_tr:nth-child(odd)]:bg-paper [&_tbody_tr:nth-child(even)]:bg-row-alt [&_tbody_tr:hover]:bg-row-hover">
      <thead>
        <tr>
          <th>{{ $t('index.age') }}</th>
          <th>{{ $t('index.message') }}</th>
          <th>{{ $t('index.author') }}</th>
          <th>{{ $t('index.commit') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="commit in commits" :key="commit.sha">
          <td class="whitespace-nowrap">{{ idleAge(commit.authored_at) }}</td>
          <td>
            <RouterLink :to="repoLocation('repo-commit', repoPath, { sha: commit.sha })">
              {{ firstLine(commit.message) }}
            </RouterLink>
          </td>
          <td class="whitespace-nowrap">{{ commit.author_name }}</td>
          <td class="font-mono whitespace-nowrap">
            <RouterLink :to="repoLocation('repo-commit', repoPath, { sha: commit.sha })">
              {{ shortSha(commit.sha) }}
            </RouterLink>
          </td>
        </tr>
        <tr v-if="!commits.length">
          <td colspan="4">{{ $t('commits.empty') }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
