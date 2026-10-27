<script setup lang="ts">
import { dataTable } from '@/lib/classes'
import { computed } from 'vue'
import { cloneUrl } from '@/api'
import PageState from '@/components/ui/PageState.vue'
import { repoPathMatches, useInjectedRepo } from '@/composables/useInjectedRepo'
import { idleAge } from '@/lib/idle'

const props = defineProps<{
  repoPath: string
}>()

const repo = useInjectedRepo()

const loading = computed(() => !repo.value || !repoPathMatches(repo.value, props.repoPath))
</script>

<template>
  <PageState :loading="loading">
    <table v-if="repo && repoPathMatches(repo, repoPath)" :class="dataTable">
      <tbody>
        <tr>
          <th>{{ $t('index.owner') }}</th>
          <td>{{ repo.owner }}</td>
        </tr>
        <tr>
          <th>{{ $t('index.description') }}</th>
          <td>{{ repo.description }}</td>
        </tr>
        <tr>
          <th>{{ $t('repo.clone') }}</th>
          <td class="font-mono">{{ cloneUrl(repoPath) }}</td>
        </tr>
        <tr>
          <th>{{ $t('index.idle') }}</th>
          <td>{{ idleAge(repo.last_activity_at || repo.updated_at) }}</td>
        </tr>
      </tbody>
    </table>
  </PageState>
</template>
