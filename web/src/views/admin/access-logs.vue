<script setup lang="ts">
import AccessLogTable from '@/components/admin/AccessLogTable.vue'
import CpLoadMore from '@/components/admin/CpLoadMore.vue'
import CpPage from '@/components/admin/CpPage.vue'
import FilterCard from '@/components/layout/FilterCard.vue'
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

const { items, total, action, ip, username, from, to, error, loading, load } = useAdminAccessLogs()
</script>

<template>
  <CpPage :title="$t('cp.accessLogs.title')" :description="$t('cp.accessLogs.description')">
    <FilterCard :cols="3">
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('accessLogs.action') }}</span>
        <select v-model="action" class="field">
          <option v-for="a in ACTIONS" :key="a || 'all'" :value="a">
            {{ a ? a : $t('accessLogs.allActions') }}
          </option>
        </select>
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('accessLogs.ip') }}</span>
        <input v-model="ip" type="search" class="field" :placeholder="$t('accessLogs.ipPlaceholder')">
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('accessLogs.username') }}</span>
        <input
          v-model="username"
          type="search"
          class="field"
          :placeholder="$t('accessLogs.usernamePlaceholder')"
        >
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('cp.accessLogs.from') }}</span>
        <input v-model="from" type="date" class="field">
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('cp.accessLogs.to') }}</span>
        <input v-model="to" type="date" class="field">
      </label>
    </FilterCard>

    <p v-if="error" class="mb-3 text-sm text-warn">{{ error }}</p>

    <PageState :loading="loading && !items.length">
      <AccessLogTable :items="items" />
      <CpLoadMore :loading="loading" :has-more="items.length < total" @load="load(false)" />
    </PageState>
  </CpPage>
</template>
