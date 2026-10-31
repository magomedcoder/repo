<script setup lang="ts">
import { computed, provide, ref, watch } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import AppFooter from '@/components/AppFooter.vue'
import AppHeader from '@/components/AppHeader.vue'
import CreateFolderModal from '@/components/CreateFolderModal.vue'
import CreateRepoModal from '@/components/CreateRepoModal.vue'
import { useAuth } from '@/composables/useAuth'

const route = useRoute()
const router = useRouter()
const auth = useAuth()
const showFolderModal = ref(false)
const showRepoModal = ref(false)
const contentKey = ref(0)

watch(
  () => auth.isAuthenticated.value,
  (ok) => {
    if (!ok) {
      showFolderModal.value = false
      showRepoModal.value = false
    }
  },
)

const folderPath = computed(() => {
  if (route.name !== 'folder') {
    return null
  }

  const raw = route.params.folderPath
  return typeof raw === 'string' && raw ? raw : null
})

function requireAuth(open: () => void) {
  if (!auth.isAuthenticated.value) {
    void router.push({
      name: 'login',
      query: { 
        redirect: route.fullPath
      },
    })
    return
  }
  open()
}

provide('openNewFolder', () => {
  requireAuth(() => {
    showFolderModal.value = true
  })
})

provide('openNewRepo', () => {
  requireAuth(() => {
    showRepoModal.value = true
  })
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
      v-if="auth.isAuthenticated.value && showFolderModal"
      @close="showFolderModal = false"
      @created="onCreated"
    />
    <CreateRepoModal
      v-if="auth.isAuthenticated.value && showRepoModal"
      :folder-path="folderPath"
      @close="showRepoModal = false"
      @created="onCreated"
    />
  </div>
</template>
