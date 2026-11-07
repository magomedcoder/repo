<script setup lang="ts">
import { RouterLink } from 'vue-router'
import CpPage from '@/components/admin/CpPage.vue'
import CpStatCard from '@/components/admin/CpStatCard.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import PageState from '@/components/ui/PageState.vue'
import Surface from '@/components/ui/Surface.vue'
import { useAdminOverview } from '@/composables/admin/useAdminOverview'
import { formatDate } from '@/i18n'

const { data, loading, error } = useAdminOverview()
</script>

<template>
  <CpPage :title="$t('cp.overview.title')" :description="$t('cp.overview.description')">
    <p v-if="error" class="text-sm text-warn">{{ error }}</p>
    <PageState :loading="loading && !data">
      <div v-if="data" class="space-y-4">
        <div class="grid gap-2 sm:grid-cols-2 xl:grid-cols-4">
          <CpStatCard
            :label="$t('cp.overview.users')"
            :value="data.users"
            icon="users"
            :to="{ name: 'users' }"
          />
          <CpStatCard
            :label="$t('cp.overview.repos')"
            :value="data.repos"
            icon="repo"
          />
          <CpStatCard
            :label="$t('cp.overview.mirrorsError')"
            :value="data.mirrors_error"
            icon="mirror"
            tone="warn"
            :to="{ name: 'admin-mirrors', query: { errors: '1' } }"
          />
          <CpStatCard
            :label="$t('cp.overview.events24h')"
            :value="data.events_24h"
            icon="list"
            :to="{ name: 'access-logs' }"
          />
        </div>

        <Surface class="overflow-hidden">
          <div class="flex items-center justify-between gap-2 border-b border-line-strong/40 px-3 py-2 sm:px-4">
            <h3 class="text-[13px] font-semibold text-ink">{{ $t('cp.overview.recent') }}</h3>
            <RouterLink
              :to="{ name: 'access-logs' }"
              class="text-[12px] text-link no-underline hover:underline"
            >
              {{ $t('cp.overview.allLogs') }}
            </RouterLink>
          </div>
          <EmptyState
            v-if="!data.recent_logs?.length"
            :message="$t('cp.overview.emptyLogs')"
          />
          <ul v-else class="divide-y divide-line-strong/30">
            <li
              v-for="row in data.recent_logs"
              :key="row.id"
              class="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-0.5 px-3 py-2 text-[12px] sm:px-4"
            >
              <div class="min-w-0">
                <span class="font-semibold text-ink">{{ row.action }}</span>
                <span v-if="row.username" class="ml-1.5 text-ink-muted">{{ row.username }}</span>
                <span v-if="row.path || row.repo" class="ml-1.5 text-ink-muted">
                  {{ row.repo || row.path }}
                </span>
              </div>
              <time class="shrink-0 text-ink-muted">{{ formatDate(row.created_at) }}</time>
            </li>
          </ul>
        </Surface>
      </div>
    </PageState>
  </CpPage>
</template>
