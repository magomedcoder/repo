<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { reposApi } from '@/api'
import type { TreeEntry } from '@/api/types'
import FileTable from '@/components/repo/FileTable.vue'
import PathCrumbs from '@/components/repo/PathCrumbs.vue'
import RepoShell from '@/components/repo/RepoShell.vue'
import PageState from '@/components/ui/PageState.vue'
import { useRepo } from '@/composables/useRepo'
import type { PageError } from '@/i18n'
import { pageError } from '@/i18n'
import { repoLocation } from '@/lib/repoNav'

const props = defineProps<{
  repoPath: string
}>()

const { repo, loading: repoLoading, error: repoError } = useRepo(() => props.repoPath)

const route = useRoute()
const tree = ref<TreeEntry[]>([])
const error = ref<PageError | string>('')
const loading = ref(true)

const refName = computed(
  () => (route.query.ref as string) || repo.value?.default_branch || 'main',
)
const dirPath = computed(() => (route.query.path as string) || '')

const parentTo = computed(() => {
  if (!dirPath.value) {
    return repoLocation('repo', props.repoPath)
  }

  const parts = dirPath.value.split('/')
  parts.pop()
  const path = parts.join('/')
  return repoLocation('repo-tree', props.repoPath, {
    query: {
      ref: refName.value,
      path: path || undefined,
    },
  })
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await reposApi.tree(props.repoPath, {
      ref: refName.value,
      path: dirPath.value,
    })
    tree.value = res.tree ?? []
  } catch (err) {
    error.value = pageError(err, 'errors.loadTree')
  } finally {
    loading.value = false
  }
}

watch(
  () => [props.repoPath, route.query.ref, route.query.path, repo.value?.default_branch],
  load,
  { immediate: true },
)
</script>

<template>
  <RepoShell 
  :repo-path="repoPath" 
    tab="tree" 
    :repo="repo" 
    :loading="repoLoading" 
    :error="repoError"
  >
    <div class="mb-3">
      <PathCrumbs 
        :repo-path="repoPath" 
        :ref-name="refName" 
        :path="dirPath"
      />
    </div>
    <PageState :loading="loading" :error="error">
      <FileTable
        :entries="tree"
        :repo-path="repoPath"
        :ref-name="refName"
        :parent-to="dirPath ? parentTo : null"
      />
    </PageState>
  </RepoShell>
</template>
