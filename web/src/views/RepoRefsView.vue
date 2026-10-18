<script setup lang="ts">
import { dataTable } from '@/lib/classes'
import { ref, watch } from 'vue'
import { reposApi } from '@/api'
import type { RefInfo } from '@/api/types'
import RepoShell from '@/components/repo/RepoShell.vue'
import PageState from '@/components/ui/PageState.vue'
import { pageError } from '@/i18n'

const props = defineProps<{
  owner: string
  repoPath: string
}>()

const branches = ref<RefInfo[]>([])
const tags = ref<RefInfo[]>([])
const error = ref('')
const loading = ref(true)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [branchRes, tagRes] = await Promise.all([
      reposApi.branches(props.owner, props.repoPath),
      reposApi.tags(props.owner, props.repoPath),
    ])
    branches.value = branchRes.branches ?? []
    tags.value = tagRes.tags ?? []
  } catch (err) {
    error.value = pageError(err, 'errors.loadRepository')
  } finally {
    loading.value = false
  }
}

watch(() => [props.owner, props.repoPath], load, { immediate: true })
</script>

<template>
  <RepoShell :owner="owner" :repo-path="repoPath" tab="refs">
    <PageState :loading="loading" :error="error">
      <table :class="dataTable">
        <thead>
          <tr>
            <th>{{ $t('index.ref') }}</th>
            <th>{{ $t('index.commit') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="branch in branches" :key="'b-' + branch.name">
            <td>refs/heads/{{ branch.name }}</td>
            <td class="font-mono">{{ branch.commit_sha }}</td>
          </tr>
          <tr v-for="tag in tags" :key="'t-' + tag.name">
            <td>refs/tags/{{ tag.name }}</td>
            <td class="font-mono">{{ tag.commit_sha }}</td>
          </tr>
        </tbody>
      </table>
    </PageState>
  </RepoShell>
</template>
