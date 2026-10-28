<script setup lang="ts">
import { ref, watch } from 'vue'
import { foldersApi } from '@/api'
import { ApiError } from '@/api/client'
import type { FolderContents } from '@/api/types'
import FolderView from '@/views/FolderView.vue'
import RepoHomeView from '@/views/RepoHomeView.vue'
import { useAuth } from '@/composables/useAuth'
import { localizeError } from '@/i18n'
import { normalizeRepoPath } from '@/lib/repoPath'

const props = defineProps<{
  folderPath: string
}>()

const auth = useAuth()
const kind = ref<'loading' | 'folder' | 'repo' | 'error'>('loading')
const error = ref('')
const folderContents = ref<FolderContents | null>(null)

async function resolve() {
  const path = normalizeRepoPath(props.folderPath)
  kind.value = 'loading'
  error.value = ''
  folderContents.value = null
  if (!path || path.includes('/')) {
    kind.value = 'repo'
    return
  }

  if (!auth.isAuthenticated.value) {
    kind.value = 'repo'
    return
  }

  try {
    folderContents.value = await foldersApi.get(path)
    kind.value = 'folder'
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) {
      kind.value = 'repo'
      return
    }
    kind.value = 'error'
    error.value = localizeError(err, 'errors.loadFailed')
  }
}

watch(() => [props.folderPath, auth.isAuthenticated.value] as const, resolve, { immediate: true })
</script>

<template>
  <p v-if="kind === 'loading'" class="text-sm text-ink-muted">{{ $t('common.loading') }}</p>
  <p v-else-if="kind === 'error'" class="text-sm text-warn">{{ error }}</p>
  <FolderView
    v-else-if="kind === 'folder'"
    :folder-path="folderPath"
    :initial="folderContents"
  />
  <RepoHomeView v-else :repo-path="folderPath" />
</template>
