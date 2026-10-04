<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { reposApi } from '@/api'
import type { Repository } from '@/api/types'
import PageState from '@/components/ui/PageState.vue'
import VisibilityBadge from '@/components/ui/VisibilityBadge.vue'
import { useBreadcrumbs } from '@/composables/useBreadcrumbs'
import { localizeError } from '@/i18n'

const route = useRoute()
const router = useRouter()
const { setBreadcrumbs, clearBreadcrumbs } = useBreadcrumbs()

const q = ref(String(route.query.q || ''))
const scope = ref<'own' | 'public'>(route.query.scope === 'public' ? 'public' : 'own')
const repos = ref<Repository[]>([])
const loading = ref(false)
const error = ref('')
let timer: ReturnType<typeof setTimeout> | null = null

async function search() {
  const query = q.value.trim()
  if (!query) {
    repos.value = []
    return
  }

  loading.value = true
  error.value = ''
  try {
    const res = await reposApi.search(query, scope.value === 'public' ? 'public' : undefined)
    repos.value = res.repos ?? []
  } catch (err) {
    error.value = localizeError(err, 'errors.searchFailed')
  } finally {
    loading.value = false
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
  const path = repo.folder_path ? `${repo.folder_path}/${repo.name}` : repo.name
  return { 
    name: 'repo' as const, 
    params: { 
      owner: repo.owner, 
      repoPath: path 
    } 
  }
}

watch(scope, schedule)

onMounted(() => {
  setBreadcrumbs([{ 
    label: 'search', 
    labelKey: 'search.title' 
  }])
  if (q.value.trim()) {
    search()
  }
})
onUnmounted(() => {
  clearBreadcrumbs()
  if (timer) {
    clearTimeout(timer)
  }
})
</script>

<template>
  <div>
    <h1 class="text-2xl font-semibold">{{ $t('search.title') }}</h1>
    <p class="mt-1 text-sm text-ink-muted">{{ $t('search.lead') }}</p>

    <div class="mt-6 flex flex-wrap items-end gap-3">
      <label class="block min-w-64 flex-1 text-sm">
        <span class="mb-1 block font-semibold">{{ $t('search.query') }}</span>
        <input
          v-model="q"
          type="search"
          class="input w-full"
          :placeholder="$t('search.placeholder')"
          @input="schedule"
        >
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('search.scope') }}</span>
        <select v-model="scope" class="input min-w-40">
          <option value="own">{{ $t('search.scopeOwn') }}</option>
          <option value="public">{{ $t('search.scopePublic') }}</option>
        </select>
      </label>
    </div>

    <PageState class="mt-6" :loading="loading" :error="error">
      <p v-if="!q.trim()" class="text-sm text-ink-muted">{{ $t('search.hint') }}</p>
      <p v-else-if="!repos.length" class="text-sm text-ink-muted">{{ $t('search.empty') }}</p>
      <ul v-else class="divide-y divide-line rounded-md border border-line bg-white">
        <li v-for="repo in repos" :key="repo.id">
          <RouterLink
            :to="repoTo(repo)"
            class="flex items-center justify-between gap-3 px-4 py-3 hover:bg-paper"
          >
            <div class="min-w-0">
              <p class="truncate font-semibold">
                {{ repo.owner }}/{{ repo.folder_path ? `${repo.folder_path}/` : '' }}{{ repo.name }}
              </p>
              <p v-if="repo.description" class="truncate text-sm text-ink-muted">{{ repo.description }}</p>
            </div>
            <VisibilityBadge :is-private="repo.is_private" />
          </RouterLink>
        </li>
      </ul>
    </PageState>
  </div>
</template>
