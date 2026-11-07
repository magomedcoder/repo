<script setup lang="ts">
import { RouterLink } from 'vue-router'
import type { CommitInfo } from '@/api/types'
import Icon from '@/components/ui/Icon.vue'
import GlassTable from '@/components/ui/GlassTable.vue'
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
  <div>
    <GlassTable>
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
          <td class="whitespace-nowrap">
            <span class="inline-flex items-center gap-1 text-ink-muted">
              <Icon name="clock" size="sm" />
              <span class="text-ink">{{ idleAge(commit.authored_at) }}</span>
            </span>
          </td>
          <td>
            <RouterLink :to="repoLocation('repo-commit', repoPath, { sha: commit.sha })" class="inline-flex items-center gap-1.5">
              <Icon name="commit" size="sm" class="text-ink-muted" />
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
    </GlassTable>
  </div>
</template>
