<script setup lang="ts">
import { textButton } from '@/lib/classes'
import { computed, inject, onUnmounted, ref, watch } from 'vue'
import { foldersApi } from '@/api'
import type { Folder, RepoSummary } from '@/api/types'
import ContentsList from '@/components/ContentsList.vue'
import CreateFolderModal from '@/components/CreateFolderModal.vue'
import CreateRepoModal from '@/components/CreateRepoModal.vue'
import { useAuth } from '@/composables/useAuth'
import { useBreadcrumbs, type Crumb } from '@/composables/useBreadcrumbs'
import { localizeError } from '@/i18n'

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

    const crumbs: Crumb[] = [{
      label: auth.user.value?.username || '',
      to: '/',
    }]
    if (folder.value?.path) {
      folder.value.path.split('/').forEach((part) => {
        crumbs.push({ label: part })
      })
    }
    setBreadcrumbs(crumbs)
  } catch (err) {
    error.value = localizeError(err, 'errors.loadFolder')
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
    <p class="mb-2 px-2">
      <span class="font-mono">{{ folder?.path || $t('folder.fallback') }}</span>
      <span class="px-1 text-ink-muted">|</span>
      <button type="button" :class="textButton" @click="showFolderModal = true">{{ $t('home.newFolder') }}</button>
      <span class="px-1 text-ink-muted">|</span>
      <button type="button" :class="textButton" @click="showRepoModal = true">{{ $t('home.newRepository') }}</button>
    </p>

    <p v-if="loading" class="text-sm text-ink-muted">{{ $t('common.loading') }}</p>
    <p v-else-if="error" class="text-sm text-warn">{{ error }}</p>
    <ContentsList
      v-else
      :folders="folders"
      :repos="repos"
      :folder-path="folder?.path || ''"
      :repo-owner="auth.user.value?.username"
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
