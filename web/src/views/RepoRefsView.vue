<script setup lang="ts">
import { ref, watch } from 'vue'
import { reposApi } from '@/api'
import type { RefInfo } from '@/api/types'
import RepoShell from '@/components/repo/RepoShell.vue'
import PageState from '@/components/ui/PageState.vue'
import { useRepo } from '@/composables/useRepo'
import type { PageError } from '@/i18n'
import { pageError } from '@/i18n'

const props = defineProps<{
  repoPath: string
}>()

const { repo, loading: repoLoading, error: repoError } = useRepo(() => props.repoPath)

const branches = ref<RefInfo[]>([])
const tags = ref<RefInfo[]>([])
const error = ref<PageError | string>('')
const loading = ref(true)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [branchRes, tagRes] = await Promise.all([
      reposApi.branches(props.repoPath),
      reposApi.tags(props.repoPath),
    ])
    branches.value = branchRes.branches ?? []
    tags.value = tagRes.tags ?? []
  } catch (err) {
    error.value = pageError(err, 'errors.loadRepository')
  } finally {
    loading.value = false
  }
}

watch(() => props.repoPath, load, { immediate: true })
</script>

<template>
  <RepoShell
    :repo-path="repoPath"
    tab="refs"
    :repo="repo"
    :loading="repoLoading"
    :error="repoError"
  >
    <PageState :loading="loading" :error="error">
      <table class="w-full border-collapse [&_th]:px-2 [&_th]:py-1 [&_th]:text-left [&_th]:align-baseline [&_th]:font-medium [&_th]:text-ink-muted [&_td]:border-0 [&_td]:px-2 [&_td]:py-1 [&_td]:align-baseline [&_tbody_tr:nth-child(odd)]:bg-paper [&_tbody_tr:nth-child(even)]:bg-row-alt [&_tbody_tr:hover]:bg-row-hover">
        <thead>
          <tr>
            <th>{{ $t('index.ref') }}</th>
            <th>{{ $t('index.commit') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="branch in branches" :key="'b-' + branch.name">
            <td>{{ $t('refs.head', { name: branch.name }) }}</td>
            <td class="font-mono">{{ branch.commit_sha }}</td>
          </tr>
          <tr v-for="tag in tags" :key="'t-' + tag.name">
            <td>{{ $t('refs.tag', { name: tag.name }) }}</td>
            <td class="font-mono">{{ tag.commit_sha }}</td>
          </tr>
        </tbody>
      </table>
    </PageState>
  </RepoShell>
</template>
