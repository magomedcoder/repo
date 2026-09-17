<script setup lang="ts">
import { inject, onMounted, onUnmounted, ref } from 'vue'
import { foldersApi, reposApi } from '@/api'
import type { Folder, Repository } from '@/api/types'
import ContentsList from '@/components/ContentsList.vue'
import CreateFolderModal from '@/components/CreateFolderModal.vue'
import CreateRepoModal from '@/components/CreateRepoModal.vue'
import { useAuth } from '@/composables/useAuth'
import { useBreadcrumbs } from '@/composables/useBreadcrumbs'
import { ApiError } from '@/api/client'

const auth = useAuth()
const { setBreadcrumbs, clearBreadcrumbs } = useBreadcrumbs()
const reloadSidebar = inject<() => void>('reloadSidebar', () => {})

const folders = ref<Folder[]>([])
const repos = ref<Repository[]>([])
const error = ref('')
const loading = ref(true)
const showFolderModal = ref(false)
const showRepoModal = ref(false)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [folderRes, repoRes] = await Promise.all([foldersApi.list(), reposApi.listOwn()])
    folders.value = folderRes.folders ?? []
    repos.value = (repoRes.repos ?? []).filter((r) => r.folder_id == null)
    setBreadcrumbs([{ label: auth.user.value?.username || 'home' }])
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

function onCreated() {
  reloadSidebar()
  load()
}

onMounted(load)
onUnmounted(clearBreadcrumbs)
</script>

<template>
  <div>
    <div class="mb-6 flex flex-wrap items-end justify-between gap-3">
      <div>
        <h1 class="font-display text-3xl font-bold">Dashboard</h1>
        <p class="mt-1 text-sm text-ink-muted">Root folders and repositories in your namespace.</p>
      </div>
      <div class="flex flex-wrap gap-2">
        <button
          type="button"
          class="btn-ghost"
          @click="showFolderModal = true"
        >New folder</button>
        <button
          type="button"
          class="btn-primary"
          @click="showRepoModal = true"
        >New repository</button>
      </div>
    </div>

    <p v-if="loading" class="text-sm text-ink-muted">Loading...</p>
    <p v-else-if="error" class="text-sm text-warn">{{ error }}</p>
    <ContentsList v-else :folders="folders" :repos="repos" />

    <CreateFolderModal v-if="showFolderModal" @close="showFolderModal = false" @created="onCreated" />
    <CreateRepoModal v-if="showRepoModal" @close="showRepoModal = false" @created="onCreated" />
  </div>
</template>
