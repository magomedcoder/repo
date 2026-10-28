<script setup lang="ts">
import { computed, provide, ref } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import AppFooter from '@/components/AppFooter.vue'
import AppHeader from '@/components/AppHeader.vue'
import CreateFolderModal from '@/components/CreateFolderModal.vue'
import CreateRepoModal from '@/components/CreateRepoModal.vue'

const route = useRoute()
const showFolderModal = ref(false)
const showRepoModal = ref(false)
const contentKey = ref(0)

const folderPath = computed(() => {
  if (route.name !== 'folder') {
    return null
  }
  
  const raw = route.params.folderPath
  return typeof raw === 'string' && raw ? raw : null
})

provide('openNewFolder', () => {
  showFolderModal.value = true
})

provide('openNewRepo', () => {
  showRepoModal.value = true
})

async function onCreated() {
  showFolderModal.value = false
  showRepoModal.value = false
  if (route.name === 'home' || route.name === 'folder' || route.name === 'folders') {
    contentKey.value += 1
  }
}
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col">
    <AppHeader />
    <main class="min-w-0 flex-1">
      <RouterView :key="contentKey" />
    </main>
    <AppFooter />

    <CreateFolderModal
      v-if="showFolderModal"
      @close="showFolderModal = false"
      @created="onCreated"
    />
    <CreateRepoModal
      v-if="showRepoModal"
      :folder-path="folderPath"
      @close="showRepoModal = false"
      @created="onCreated"
    />
  </div>
</template>
