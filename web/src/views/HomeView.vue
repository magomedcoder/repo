<script setup lang="ts">
import { inject, onMounted, onUnmounted, ref } from 'vue'
import { foldersApi, reposApi } from '@/api'
import type { Folder, Repository } from '@/api/types'
import ContentsList from '@/components/ContentsList.vue'
import CreateFolderModal from '@/components/CreateFolderModal.vue'
import CreateRepoModal from '@/components/CreateRepoModal.vue'
import { useAuth } from '@/composables/useAuth'
import { useBreadcrumbs } from '@/composables/useBreadcrumbs'
import { localizeError } from '@/i18n'

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
    setBreadcrumbs([{
      label: auth.user.value?.username || 'home'
    }])
  } catch (err) {
    error.value = localizeError(err, 'errors.loadFailed')
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
        <h1 class="text-2xl font-semibold">{{ $t('home.title') }}</h1>
        <p class="mt-1 text-sm text-ink-muted">{{ $t('home.lead') }}</p>
      </div>
      <div class="flex flex-wrap gap-2">
        <button
          type="button"
          class="btn-ghost"
          @click="showFolderModal = true"
        >{{ $t('home.newFolder') }}</button>
        <button
          type="button"
          class="btn-primary"
          @click="showRepoModal = true"
        >{{ $t('home.newRepository') }}</button>
      </div>
    </div>

    <p v-if="loading" class="text-sm text-ink-muted">{{ $t('common.loading') }}</p>
    <p v-else-if="error" class="text-sm text-warn">{{ error }}</p>
    <ContentsList
      v-else
      :folders="folders"
      :repos="repos"
    />

    <CreateFolderModal
      v-if="showFolderModal"
      @close="showFolderModal = false"
      @created="onCreated"
    />
    <CreateRepoModal
      v-if="showRepoModal"
      @close="showRepoModal = false"
      @created="onCreated"
    />
  </div>
</template>
