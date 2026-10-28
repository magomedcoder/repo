<script setup lang="ts">
import { btnGhost } from '@/lib/classes'
import { onMounted, ref, watch } from 'vue'
import { foldersApi, reposApi } from '@/api'
import type { Folder, Repository } from '@/api/types'
import ContentsList from '@/components/ContentsList.vue'
import { useAuth } from '@/composables/useAuth'
import { localizeError } from '@/i18n'

const PAGE_SIZE = 50

const auth = useAuth()
const ownRepos = ref<Repository[]>([])
const homeRepos = ref<Repository[]>([])
const folders = ref<Folder[]>([])
const error = ref('')
const loading = ref(true)
const loadingMoreOwn = ref(false)
const loadingMoreHome = ref(false)
const hasMoreOwn = ref(false)
const hasMoreHome = ref(false)
const sort = ref('name')

async function loadOwn(append = false) {
  if (!auth.isAuthenticated.value) {
    ownRepos.value = []
    folders.value = []
    hasMoreOwn.value = false
    return
  }
  
  if (append) {
    loadingMoreOwn.value = true
  }

  try {
    const offset = append ? ownRepos.value.length : 0
    const [repoRes, folderRes] = await Promise.all([
      reposApi.listOwn(sort.value, { offset, limit: PAGE_SIZE }),
      append ? Promise.resolve(null) : foldersApi.list().catch(() => ({ folders: [] as Folder[] })),
    ])
    const batch = repoRes.repos ?? []
    ownRepos.value = append ? [...ownRepos.value, ...batch] : batch
    hasMoreOwn.value = batch.length >= PAGE_SIZE
    if (!append && folderRes) {
      folders.value = folderRes.folders ?? []
    }
  } catch (err) {
    error.value = localizeError(err, 'errors.loadFailed')
  } finally {
    loadingMoreOwn.value = false
  }
}

async function loadHome(append = false) {
  if (append) {
    loadingMoreHome.value = true
  }
  try {
    const offset = append ? homeRepos.value.length : 0
    const repoRes = await reposApi.listHome(sort.value, { offset, limit: PAGE_SIZE })
    const batch = repoRes.repos ?? []
    homeRepos.value = append ? [...homeRepos.value, ...batch] : batch
    hasMoreHome.value = batch.length >= PAGE_SIZE
  } catch (err) {
    error.value = localizeError(err, 'errors.loadFailed')
  } finally {
    loadingMoreHome.value = false
  }
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    await Promise.all([loadOwn(false), loadHome(false)])
  } finally {
    loading.value = false
  }
}

function onSort(key: string) {
  sort.value = key
  load()
}

watch(
  () => auth.isAuthenticated.value,
  () => {
    load()
  },
)

onMounted(() => load())
</script>

<template>
  <div>
    <p v-if="loading" class="text-ink-muted">{{ $t('common.loading') }}</p>
    <p v-else-if="error" class="text-warn">{{ error }}</p>
    <template v-else>
      <section v-if="auth.isAuthenticated.value" class="mb-8">
        <h2 class="mb-2 text-sm font-semibold text-ink">{{ $t('home.myRepos') }}</h2>
        <ContentsList
          :folders="folders"
          :repos="ownRepos"
          show-folder
          @sort="onSort"
        />
        <p v-if="hasMoreOwn" class="mt-3">
          <button
            type="button"
            :class="btnGhost"
            :disabled="loadingMoreOwn"
            @click="loadOwn(true)"
          >
            {{ loadingMoreOwn ? $t('common.loading') : $t('common.loadMore') }}
          </button>
        </p>
      </section>

      <section>
        <h2 class="mb-2 text-sm font-semibold text-ink">{{ $t('home.publicRepos') }}</h2>
        <ContentsList
          :repos="homeRepos"
          show-folder
          @sort="onSort"
        />
        <p v-if="hasMoreHome" class="mt-3">
          <button
            type="button"
            :class="btnGhost"
            :disabled="loadingMoreHome"
            @click="loadHome(true)"
          >
            {{ loadingMoreHome ? $t('common.loading') : $t('common.loadMore') }}
          </button>
        </p>
        <p
          v-else-if="!homeRepos.length"
          class="text-sm text-ink-muted"
        >
          {{ $t('home.publicEmpty') }}
        </p>
      </section>
    </template>
  </div>
</template>
