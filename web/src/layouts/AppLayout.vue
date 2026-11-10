<script setup lang="ts">
import { provide, ref, watch } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import AppFooter from '@/components/layout/AppFooter.vue'
import AppHeader from '@/components/layout/AppHeader.vue'
import CreateRepoModal from '@/components/modals/CreateRepoModal.vue'
import ImportRepoModal from '@/components/modals/ImportRepoModal.vue'
import MirrorRepoModal from '@/components/modals/MirrorRepoModal.vue'
import { useAuth } from '@/composables/useAuth'
import { useRouteSeo } from '@/composables/useRouteSeo'

type RepoModalKind = 'create' | 'import' | 'mirror' | null

const route = useRoute()
const router = useRouter()
const auth = useAuth()
useRouteSeo()
const repoModal = ref<RepoModalKind>(null)
const repoPrefillURL = ref('')
const contentKey = ref(0)

watch(
  () => auth.isAuthenticated.value,
  (ok) => {
    if (!ok) {
      repoModal.value = null
    }
  },
)

function requireAuth(open: () => void) {
  if (!auth.isAuthenticated.value) {
    void router.push({
      name: 'login',
      query: {
        redirect: route.fullPath,
      },
    })
    return
  }
  open()
}

provide('openNewRepo', () => {
  requireAuth(() => {
    repoPrefillURL.value = ''
    repoModal.value = 'create'
  })
})

provide('openImportRepo', (url?: string) => {
  requireAuth(() => {
    repoPrefillURL.value = url ?? ''
    repoModal.value = 'import'
  })
})

provide('openMirrorRepo', (url?: string) => {
  requireAuth(() => {
    repoPrefillURL.value = url ?? ''
    repoModal.value = 'mirror'
  })
})

watch(
  () => [route.query.import_url, route.query.mirror_url, auth.isAuthenticated.value] as const,
  ([importURL, mirrorURL, ok]) => {
    if (!ok) {
      return
    }
    
    if (typeof importURL === 'string' && importURL) {
      repoPrefillURL.value = importURL
      repoModal.value = 'import'
      return
    }

    if (typeof mirrorURL === 'string' && mirrorURL) {
      repoPrefillURL.value = mirrorURL
      repoModal.value = 'mirror'
    }
  },
  { immediate: true },
)

async function onCreated() {
  repoModal.value = null
  repoPrefillURL.value = ''
  if (route.name === 'home') {
    contentKey.value += 1
  }
}

function closeRepoModal() {
  repoModal.value = null
  repoPrefillURL.value = ''
}
</script>

<template>
  <div class="app-shell">
    <AppHeader />
    <main class="min-w-0 flex-1">
      <div class="content-block">
        <RouterView :key="contentKey" />
      </div>
    </main>
    <AppFooter />

    <CreateRepoModal
      v-if="auth.isAuthenticated.value && repoModal === 'create'"
      @close="closeRepoModal"
      @created="onCreated"
    />
    <ImportRepoModal
      v-if="auth.isAuthenticated.value && repoModal === 'import'"
      :initial-url="repoPrefillURL"
      @close="closeRepoModal"
      @created="onCreated"
    />
    <MirrorRepoModal
      v-if="auth.isAuthenticated.value && repoModal === 'mirror'"
      :initial-url="repoPrefillURL"
      @close="closeRepoModal"
      @created="onCreated"
    />
  </div>
</template>
