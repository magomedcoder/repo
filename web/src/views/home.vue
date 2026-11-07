<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { reposApi } from '@/api'
import type { Repository } from '@/api/types'
import ContentsList from '@/components/ContentsList.vue'
import { useAuth } from '@/composables/useAuth'
import { localizeError } from '@/i18n'
import Icon from '@/components/ui/Icon.vue'

const PAGE_SIZE = 50

const auth = useAuth()

const ownRepos = ref<Repository[]>([])
const homeRepos = ref<Repository[]>([])
const error = ref('')
const loading = ref(true)
const loadingMore = ref(false)
const hasMoreOwn = ref(false)
const hasMoreHome = ref(false)
const sort = ref('idle')

const repos = computed(() => {
  if (!auth.isAuthenticated.value) {
    return homeRepos.value
  }

  const seen = new Set(ownRepos.value.map((r) => r.id))
  return [...ownRepos.value, ...homeRepos.value.filter((r) => !seen.has(r.id))]
})

const hasMore = computed(() => auth.isAuthenticated.value ? hasMoreOwn.value || hasMoreHome.value : hasMoreHome.value)

async function loadOwn(append = false) {
  if (!auth.isAuthenticated.value) {
    ownRepos.value = []
    hasMoreOwn.value = false
    return
  }

  try {
    const offset = append ? ownRepos.value.length : 0
    const repoRes = await reposApi.listOwn(sort.value, { offset, limit: PAGE_SIZE })
    const batch = repoRes.repos ?? []
    ownRepos.value = append ? [...ownRepos.value, ...batch] : batch
    hasMoreOwn.value = batch.length >= PAGE_SIZE
  } catch (err) {
    error.value = localizeError(err, 'errors.loadFailed')
  }
}

async function loadHome(append = false) {
  try {
    const offset = append ? homeRepos.value.length : 0
    const repoRes = await reposApi.listHome(sort.value, { offset, limit: PAGE_SIZE })
    const batch = repoRes.repos ?? []
    homeRepos.value = append ? [...homeRepos.value, ...batch] : batch
    hasMoreHome.value = batch.length >= PAGE_SIZE
  } catch (err) {
    error.value = localizeError(err, 'errors.loadFailed')
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

async function loadMore() {
  if (loadingMore.value || !hasMore.value) {
    return
  }

  loadingMore.value = true
  try {
    const tasks: Promise<void>[] = []
    if (auth.isAuthenticated.value && hasMoreOwn.value) {
      tasks.push(loadOwn(true))
    }
    
    if (hasMoreHome.value) {
      tasks.push(loadHome(true))
    }

    await Promise.all(tasks)
  } finally {
    loadingMore.value = false
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
  <div class="page-enter">
    <p v-if="loading" class="text-ink-muted">{{ $t('common.loading') }}</p>
    <p v-else-if="error" class="text-warn">{{ error }}</p>
    <template v-else>
      <ContentsList
        :repos="repos"
        :sort="sort"
        :title="$t('home.pinned')"
        :hint="$t('home.pinnedHint')"
        :empty-text="$t('home.emptyPinned')"
        show-folder
        @sort="onSort"
      />
      <p v-if="hasMore" class="mt-3">
        <button
          type="button"
          class="btn"
          :disabled="loadingMore"
          @click="loadMore"
        >
          <Icon name="chevron-down" size="sm" />
          {{ loadingMore ? $t('common.loading') : $t('common.loadMore') }}
        </button>
      </p>
    </template>
  </div>
</template>
