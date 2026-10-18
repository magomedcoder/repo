<script setup lang="ts">
import { useRepoOwner } from '@/composables/useRepoOwner'
import { dataTable } from '@/lib/classes'
import { ref, watch } from 'vue'
import { cloneUrl, reposApi } from '@/api'
import type { Repository } from '@/api/types'
import RepoShell from '@/components/repo/RepoShell.vue'
import PageState from '@/components/ui/PageState.vue'
import { idleAge } from '@/lib/idle'
import { pageError } from '@/i18n'

const owner = useRepoOwner()

const props = defineProps<{
  repoPath: string
}>()

const repo = ref<Repository | null>(null)
const error = ref('')
const loading = ref(true)

async function load() {
  loading.value = true
  error.value = ''
  try {
    repo.value = await reposApi.get(owner.value, props.repoPath)
  } catch (err) {
    error.value = pageError(err, 'errors.loadRepository')
  } finally {
    loading.value = false
  }
}

watch(() => [owner.value, props.repoPath], load, { immediate: true })
</script>

<template>
  <RepoShell :owner="owner" :repo-path="repoPath" tab="about">
    <PageState :loading="loading" :error="error">
      <table v-if="repo" :class="dataTable">
        <tbody>
          <tr>
            <th>{{ $t('index.owner') }}</th>
            <td>{{ owner }}</td>
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
  </RepoShell>
</template>
