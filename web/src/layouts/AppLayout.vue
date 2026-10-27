<script setup lang="ts">
import { computed, provide, ref } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import AppFooter from '@/components/AppFooter.vue'
import AppHeader from '@/components/AppHeader.vue'
import CreateFolderModal from '@/components/CreateFolderModal.vue'
import CreateRepoModal from '@/components/CreateRepoModal.vue'

const route = useRoute()
const router = useRouter()
const showFolderModal = ref(false)
const showRepoModal = ref(false)

const folderId = computed(() =>
  route.name === 'folder' ? Number(route.params.id) : null,
)

provide('openNewFolder', () => {
  showFolderModal.value = true
})

provide('openNewRepo', () => {
  showRepoModal.value = true
})

async function onCreated() {
  showFolderModal.value = false
  showRepoModal.value = false
  if (route.name === 'home' || route.name === 'folder') {
    await router.go(0)
  }
}
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col">
    <AppHeader />
    <main class="min-w-0 flex-1">
      <RouterView />
    </main>
    <AppFooter />

    <CreateFolderModal
      v-if="showFolderModal"
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
