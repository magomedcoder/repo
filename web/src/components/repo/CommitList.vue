<script setup lang="ts">
import { dataTable } from '@/lib/classes'
import { RouterLink } from 'vue-router'
import type { CommitInfo } from '@/api/types'
import { idleAge } from '@/lib/idle'

defineProps<{
  commits: CommitInfo[]
  owner: string
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
    <table :class="dataTable">
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
            <RouterLink
              :to="{
                name: 'repo-commit',
                params: { repoPath, sha: commit.sha },
              }"
            >
              {{ firstLine(commit.message) }}
            </RouterLink>
          </td>
          <td class="whitespace-nowrap">{{ commit.author_name }}</td>
          <td class="font-mono whitespace-nowrap">
            <RouterLink
              :to="{
                name: 'repo-commit',
                params: { repoPath, sha: commit.sha },
              }"
            >
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
