<script setup lang="ts">
import { dataTable, field } from '@/lib/classes'
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { reposApi } from '@/api'
import type { Repository } from '@/api/types'
import PageState from '@/components/ui/PageState.vue'
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
    <div class="flex flex-wrap items-end gap-3 px-2">
      <label class="block min-w-64 flex-1 text-sm">
        <span class="mb-1 block font-semibold">{{ $t('search.query') }}</span>
        <input
          v-model="q"
          type="search"
          :class="[field, 'w-full']"
          :placeholder="$t('search.placeholder')"
          @input="schedule"
        >
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('search.scope') }}</span>
        <select v-model="scope" :class="[field, 'min-w-40']">
          <option value="own">{{ $t('search.scopeOwn') }}</option>
          <option value="public">{{ $t('search.scopePublic') }}</option>
        </select>
      </label>
    </div>

    <PageState class="mt-2" :loading="loading" :error="error">
      <p v-if="!q.trim()" class="px-2">{{ $t('search.hint') }}</p>
      <p v-else-if="!repos.length" class="px-2">{{ $t('search.empty') }}</p>
      <table v-else :class="dataTable">
        <thead>
          <tr>
            <th>{{ $t('index.name') }}</th>
            <th>{{ $t('index.description') }}</th>
            <th>{{ $t('index.owner') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="repo in repos" :key="repo.id">
            <td>
              <RouterLink :to="repoTo(repo)">
                {{ repo.folder_path ? `${repo.folder_path}/` : '' }}{{ repo.name }}
              </RouterLink>
            </td>
            <td>{{ repo.description }}</td>
            <td>{{ repo.owner }}</td>
          </tr>
        </tbody>
      </table>
    </PageState>
  </div>
</template>
