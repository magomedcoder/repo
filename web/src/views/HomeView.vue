<script setup lang="ts">
import { textButton } from '@/lib/classes'
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
const sort = ref('name')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [repoRes, folderRes] = await Promise.all([
      reposApi.listOwn(sort.value),
      foldersApi.list(),
    ])
    repos.value = repoRes.repos ?? []
    folders.value = folderRes.folders ?? []
    setBreadcrumbs([{
      label: auth.user.value?.username || 'home'
    }])
  } catch (err) {
    error.value = localizeError(err, 'errors.loadFailed')
  } finally {
    loading.value = false
  }
}

function onSort(key: string) {
  sort.value = key
  load()
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
    <p class="mb-2 px-2">
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
      group-by-path
      @sort="onSort"
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
