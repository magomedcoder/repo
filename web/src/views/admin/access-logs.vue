<script setup lang="ts">
import AccessLogTable from '@/components/admin/AccessLogTable.vue'
import CpPage from '@/components/admin/CpPage.vue'
import CpPagination from '@/components/admin/CpPagination.vue'
import PageState from '@/components/ui/PageState.vue'
import { useAdminAccessLogs } from '@/composables/admin/useAdminAccessLogs'

const ACTIONS = [
  '',
  'visit',
  'clone',
  'push',
  'download',
  'login',
  'logout',
  'register',
  'failed_auth',
  'token_create',
  'token_revoke',
  'repo_create',
  'repo_delete',
  'repo_move',
  'admin_user_create',
  'admin_user_update',
  'admin_user_delete',
] as const

const {
  items,
  total,
  page,
  pageSize,
  action,
  ip,
  username,
  from,
  to,
  error,
  loading,
  setPage,
} = useAdminAccessLogs()
</script>

<template>
  <CpPage :title="$t('cp.accessLogs.title')" :description="$t('cp.accessLogs.description')">
    <div class="mb-3 flex flex-wrap gap-2">
      <select v-model="action" class="field w-auto min-w-[9rem]">
        <option v-for="a in ACTIONS" :key="a || 'all'" :value="a">
          {{ a ? a : $t('accessLogs.allActions') }}
        </option>
      </select>
      <input
        v-model="username"
        type="search"
        class="field min-w-[10rem] flex-1"
        :placeholder="$t('accessLogs.usernamePlaceholder')"
      >
      <input
        v-model="ip"
        type="search"
        class="field min-w-[9rem] flex-1"
        :placeholder="$t('accessLogs.ipPlaceholder')"
      >
      <input v-model="from" type="date" class="field w-auto" :aria-label="$t('cp.accessLogs.from')">
      <input v-model="to" type="date" class="field w-auto" :aria-label="$t('cp.accessLogs.to')">
    </div>

    <p v-if="error" class="mb-3 text-sm text-warn">{{ error }}</p>

    <PageState :loading="loading && !items.length">
      <AccessLogTable :items="items" />
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
