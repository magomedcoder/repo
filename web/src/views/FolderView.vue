<script setup lang="ts">
import { computed, inject, onUnmounted, ref, watch } from 'vue'
import { foldersApi } from '@/api'
import type { Folder, RepoSummary } from '@/api/types'
import ContentsList from '@/components/ContentsList.vue'
import CreateFolderModal from '@/components/CreateFolderModal.vue'
import CreateRepoModal from '@/components/CreateRepoModal.vue'
import { useAuth } from '@/composables/useAuth'
import { useBreadcrumbs, type Crumb } from '@/composables/useBreadcrumbs'
import { ApiError } from '@/api/client'

const props = defineProps<{ id: string }>()

const auth = useAuth()
const { setBreadcrumbs, clearBreadcrumbs } = useBreadcrumbs()
const reloadSidebar = inject<() => void>('reloadSidebar', () => {})

const folder = ref<Folder | null>(null)
const folders = ref<Folder[]>([])
const repos = ref<RepoSummary[]>([])
const error = ref('')
const loading = ref(true)
const showFolderModal = ref(false)
const showRepoModal = ref(false)

const folderId = computed(() => Number(props.id))

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await foldersApi.get(folderId.value)
    folder.value = res.folder ?? null
    folders.value = res.folders ?? []
    repos.value = res.repos ?? []

    const owner = auth.user.value?.username || ''
    const crumbs: Crumb[] = [{ label: owner, to: '/' }]
    if (folder.value?.path) {
      folder.value.path.split('/').forEach((part) => {
        crumbs.push({ label: part })
      })
    }
    setBreadcrumbs(crumbs)
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'Failed to load folder'
  } finally {
    loading.value = false
  }
}

function onCreated() {
  reloadSidebar()
  load()
}

watch(() => props.id, load, { immediate: true })
onUnmounted(clearBreadcrumbs)
</script>

<template>
  <div>
    <div class="mb-6 flex flex-wrap items-end justify-between gap-3">
      <div>
        <h1 class="font-display text-3xl font-bold">{{ folder?.name || 'Folder' }}</h1>
        <p v-if="folder" class="mt-1 font-mono text-sm text-ink-muted">{{ folder.path }}</p>
      </div>
      <div class="flex flex-wrap gap-2">
        <button type="button" class="btn-ghost" @click="showFolderModal = true">New folder</button>
        <button type="button" class="btn-primary" @click="showRepoModal = true">New repository</button>
      </div>
    </div>

    <p v-if="loading" class="text-sm text-ink-muted">Loading...</p>
    <p v-else-if="error" class="text-sm text-warn">{{ error }}</p>
    <ContentsList
      v-else
      :folders="folders"
      :repos="repos"
      :folder-path="folder?.path || ''"
    />

    <CreateFolderModal
      v-if="showFolderModal"
      :parent-id="folderId"
      @close="showFolderModal = false"
      @created="onCreated"
    />
    <CreateRepoModal
      v-if="showRepoModal"
      :folder-id="folderId"
      @close="showRepoModal = false"
      @created="onCreated"
    />
  </div>
</template>
