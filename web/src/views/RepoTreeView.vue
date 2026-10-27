<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { reposApi } from '@/api'
import type { TreeEntry } from '@/api/types'
import FileTable from '@/components/repo/FileTable.vue'
import RepoShell from '@/components/repo/RepoShell.vue'
import PageState from '@/components/ui/PageState.vue'
import type { PageError } from '@/i18n'
import { pageError } from '@/i18n'


const props = defineProps<{
  repoPath: string
}>()

const route = useRoute()
const tree = ref<TreeEntry[]>([])
const error = ref<PageError | string>('')
const loading = ref(true)

const refName = computed(() => (route.query.ref as string) || 'main')
const dirPath = computed(() => (route.query.path as string) || '')

const parentTo = computed(() => {
  if (!dirPath.value) {
    return {
      name: 'repo' as const,
      params: {
        repoPath: props.repoPath
      }
    }
  }

  const parts = dirPath.value.split('/')
  parts.pop()
  const path = parts.join('/')
  return {
    name: 'repo-tree' as const,
    params: {
      repoPath: props.repoPath
    },
    query: {
      ref: refName.value,
      path: path || undefined
    },
  }
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

watch(() => [props.repoPath, route.query.ref, route.query.path], load, { immediate: true })
</script>

<template>
  <RepoShell
    :repo-path="repoPath" tab="code">
    <p v-if="dirPath" class="mb-3 text-sm text-ink-muted">{{ refName }} / {{ dirPath }}</p>
    <PageState :loading="loading" :error="error">
      <FileTable
        :entries="tree"
        :repo-path="repoPath"
        :ref-name="refName"
        :parent-to="parentTo"
      />
    </PageState>
  </RepoShell>
</template>
