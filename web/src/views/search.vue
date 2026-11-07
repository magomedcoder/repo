<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { reposApi } from '@/api'
import type { Repository } from '@/api/types'
import PageState from '@/components/ui/PageState.vue'
import { useAuth } from '@/composables/useAuth'
import { localizeError } from '@/i18n'
import { repoLocation } from '@/lib/repoNav'
import { repositoryPath } from '@/lib/repoPath'
import Icon from '@/components/ui/Icon.vue'
import GlassTable from '@/components/ui/GlassTable.vue'
import FilterCard from '@/components/layout/FilterCard.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuth()
const q = ref(String(route.query.q || ''))
const scope = ref<'own' | 'public'>(route.query.scope === 'public' || !auth.isAuthenticated.value ? 'public' : 'own')
const PAGE_SIZE = 50

const repos = ref<Repository[]>([])
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(false)
const error = ref('')
let timer: ReturnType<typeof setTimeout> | null = null
let abort: AbortController | null = null

async function search(append = false) {
  const query = q.value.trim()
  if (!query) {
    repos.value = []
    hasMore.value = false
    return
  }

  if (!append) {
    abort?.abort()
    abort = new AbortController()
  }
  const signal = abort?.signal

  if (append) {
    loadingMore.value = true
  } else {
    loading.value = true
  }
  
  error.value = ''
  try {
    const offset = append ? repos.value.length : 0
    const usePublic = !auth.isAuthenticated.value || scope.value === 'public'
    const res = await reposApi.search(
      query,
      usePublic ? 'public' : undefined,
      { offset, limit: PAGE_SIZE },
      signal ? { signal } : undefined,
    )
    if (signal?.aborted) {
      return
    }
    const batch = res.repos ?? []
    repos.value = append ? [...repos.value, ...batch] : batch
    hasMore.value = batch.length >= PAGE_SIZE
  } catch (err) {
    if (signal?.aborted) {
      return
    }
    error.value = localizeError(err, 'errors.searchFailed')
  } finally {
    if (!signal?.aborted) {
      loading.value = false
      loadingMore.value = false
    }
  }
}

function loadMore() {
  if (!loadingMore.value && hasMore.value && q.value.trim()) {
    search(true)
  }
}

function schedule() {
  if (timer) {
    clearTimeout(timer)
  }

  timer = setTimeout(() => {
    router.replace({
      name: 'search',
      query: {
        q: q.value.trim() || undefined,
        scope: scope.value === 'public' ? 'public' : undefined,
      },
    })
    search()
  }, 250)
}

function repoTo(repo: Repository) {
  return repoLocation('repo', repositoryPath(repo))
}

watch(scope, schedule)
watch(() => route.query.q, (v) => {
  const next = typeof v === 'string' ? v : ''
  if (next !== q.value) {
    q.value = next
  }
})

onMounted(() => {
  if (q.value.trim()) {
    search()
  }
})
onUnmounted(() => {
  if (timer) clearTimeout(timer)
  abort?.abort()
})
</script>

<template>
  <div>
    <FilterCard class="flex flex-wrap items-end gap-3">
      <label class="block min-w-64 flex-1 text-sm">
        <span class="mb-1 inline-flex items-center gap-1 text-[11px] text-ink-muted"><Icon name="search" size="sm" />{{ $t('search.query') }}</span>
        <input
          v-model="q"
          :class="['field', 'w-full']"
          :placeholder="$t('search.placeholder')"
          @input="schedule"
        />
      </label>
      <label v-if="auth.isAuthenticated.value" class="block text-sm">
        <span class="mb-1 block text-[11px] text-ink-muted">{{ $t('search.scope') }}</span>
        <select v-model="scope" :class="['field', 'min-w-40']">
          <option value="own">{{ $t('search.scopeOwn') }}</option>
          <option value="public">{{ $t('search.scopePublic') }}</option>
        </select>
      </label>
    </FilterCard>
    <p v-if="!q.trim()" class="mt-3 text-sm text-ink-muted">{{ $t('search.hint') }}</p>
    <PageState class="mt-2" :loading="loading" :error="error">
      <GlassTable v-if="repos.length">
        <thead>
          <tr>
            <th>{{ $t('index.name') }}</th>
            <th>{{ $t('index.description') }}</th>
            <th>{{ $t('index.owner') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="repo in repos" :key="repo.id">
            <td><RouterLink :to="repoTo(repo)">{{ repo.name }}</RouterLink></td>
            <td>{{ repo.description }}</td>
            <td>{{ repo.owner }}</td>
          </tr>
        </tbody>
      </GlassTable>
      <p v-else-if="q.trim()" class="text-sm text-ink-muted">{{ $t('search.empty') }}</p>
      <p v-if="hasMore && repos.length" class="mt-3">
        <button type="button" class="btn" :disabled="loadingMore" @click="loadMore">
          {{ loadingMore ? $t('common.loading') : $t('common.loadMore') }}
        </button>
      </p>
    </PageState>
  </div>
</template>
