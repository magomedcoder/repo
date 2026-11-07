<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { adminReposApi } from '@/api'
import type { AdminRepo } from '@/api/types/admin'
import CpPage from '@/components/admin/CpPage.vue'
import CpPagination from '@/components/admin/CpPagination.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import PageState from '@/components/ui/PageState.vue'
import Surface from '@/components/ui/Surface.vue'
import { formatDate, localizeError } from '@/i18n'
import { repoLocation } from '@/lib/repoNav'

const repos = ref<AdminRepo[]>([])
const total = ref(0)
const q = ref('')
const privacy = ref<'' | '1' | '0'>('')
const mirrorOnly = ref(false)
const loading = ref(true)
const error = ref('')
const page = ref(1)
const pageSize = 50

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await adminReposApi.list({
      q: q.value.trim() || undefined,
      private: privacy.value === '' ? undefined : privacy.value === '1',
      mirror: mirrorOnly.value || undefined,
      limit: pageSize,
      offset: (page.value - 1) * pageSize,
    })
    total.value = res.total
    repos.value = res.repos ?? []
    const maxPage = Math.max(1, Math.ceil((res.total || 0) / pageSize) || 1)
    if (page.value > maxPage) {
      page.value = maxPage
      await load()
      return
    }
  } catch (err) {
    error.value = localizeError(err, 'errors.loadFailed')
  } finally {
    loading.value = false
  }
}

async function setPage(next: number) {
  page.value = next
  await load()
}

let timer: ReturnType<typeof setTimeout> | undefined
watch([q, privacy, mirrorOnly], () => {
  clearTimeout(timer)
  timer = setTimeout(() => {
    page.value = 1
    void load()
  }, 250)
})

onMounted(() => {
  void load()
})
</script>

<template>
  <CpPage :title="$t('cp.repos.title')" :description="$t('cp.repos.description')">
    <div class="mb-3 flex flex-wrap gap-2">
      <input v-model="q" type="search" class="field min-w-[12rem] flex-1" :placeholder="$t('cp.repos.search')">
      <select v-model="privacy" class="field w-auto">
        <option value="">{{ $t('cp.repos.allVisibility') }}</option>
        <option value="1">{{ $t('home.private') }}</option>
        <option value="0">{{ $t('cp.repos.public') }}</option>
      </select>
      <label class="inline-flex items-center gap-2 text-sm text-ink-muted">
        <input v-model="mirrorOnly" type="checkbox" class="accent-ink">
        {{ $t('cp.repos.mirrorsOnly') }}
      </label>
    </div>

    <p v-if="error" class="mb-3 text-sm text-warn">{{ error }}</p>

    <PageState :loading="loading && !repos.length">
      <Surface class="overflow-x-auto">
        <table class="w-full min-w-[40rem] border-collapse text-left text-[12px]">
          <thead class="sticky top-0 bg-white/55 text-ink backdrop-blur-sm">
            <tr>
              <th class="border-b border-line px-3 py-2 font-semibold">{{ $t('cp.repos.repository') }}</th>
              <th class="border-b border-line px-3 py-2 font-semibold">{{ $t('cp.repos.owner') }}</th>
              <th class="border-b border-line px-3 py-2 font-semibold">{{ $t('cp.repos.visibility') }}</th>
              <th class="border-b border-line px-3 py-2 font-semibold">{{ $t('cp.nav.mirrors') }}</th>
              <th class="border-b border-line px-3 py-2 font-semibold">{{ $t('index.idle') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="repo in repos" :key="repo.id" class="odd:bg-transparent even:bg-row-alt">
              <td class="border-b border-line px-3 py-2">
                <RouterLink
                  :to="repoLocation('repo', repo.repo_path)"
                  class="font-semibold text-link no-underline hover:underline"
                >
                  {{ repo.name }}
                </RouterLink>
                <p v-if="repo.description" class="mt-0.5 line-clamp-1 text-[11px] text-ink-muted">
                  {{ repo.description }}
                </p>
              </td>
              <td class="border-b border-line px-3 py-2 text-ink-muted">{{ repo.owner || '-' }}</td>
              <td class="border-b border-line px-3 py-2">
                <span class="inline-flex rounded-full border border-line-strong/70 px-1.5 py-px text-[10px] text-ink-muted">
                  {{ repo.is_private ? $t('home.private') : $t('cp.repos.public') }}
                </span>
              </td>
              <td class="border-b border-line px-3 py-2 text-ink-muted">
                <template v-if="repo.mirror_direction">
                  {{ repo.mirror_direction }}
                  <span v-if="repo.mirror_status"> {{ repo.mirror_status }}</span>
                </template>
                <template v-else>-</template>
              </td>
              <td class="border-b border-line px-3 py-2 whitespace-nowrap text-ink-muted">
                {{ formatDate(repo.last_activity_at || repo.updated_at) }}
              </td>
            </tr>
          </tbody>
        </table>
        <EmptyState v-if="!repos.length" :message="$t('cp.repos.empty')" />
      </Surface>
      <CpPagination
        :page="page"
        :page-size="pageSize"
        :total="total"
        :loading="loading"
        @update:page="setPage"
      />
    </PageState>
  </CpPage>
</template>
