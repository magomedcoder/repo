<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { adminAccessLogsApi } from '@/api'
import type { AccessLog } from '@/api/types'
import AccessLogTable from '@/components/admin/AccessLogTable.vue'
import PageState from '@/components/ui/PageState.vue'
import { localizeError } from '@/i18n'
import PageHeader from '@/components/ui/PageHeader.vue'
import FilterCard from '@/components/layout/FilterCard.vue'

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

const items = ref<AccessLog[]>([])
const total = ref(0)
const action = ref('')
const ip = ref('')
const username = ref('')
const error = ref('')
const loading = ref(true)
const pageSize = 50

async function load(reset = true) {
  loading.value = true
  error.value = ''
  try {
    const offset = reset ? 0 : items.value.length
    const res = await adminAccessLogsApi.list({
      action: action.value || undefined,
      ip: ip.value.trim() || undefined,
      username: username.value.trim() || undefined,
      limit: pageSize,
      offset,
    })
    total.value = res.total
    items.value = reset ? (res.items ?? []) : [...items.value, ...(res.items ?? [])]
  } catch (err) {
    error.value = localizeError(err, 'errors.loadAccessLogs')
  } finally {
    loading.value = false
  }
}

let timer: ReturnType<typeof setTimeout> | undefined
watch([action, ip, username], () => {
  clearTimeout(timer)
  timer = setTimeout(() => {
    void load(true)
  }, 250)
})

onMounted(() => {
  void load(true)
})
</script>

<template>
  <div class="mx-auto max-w-5xl">
    <PageHeader :title="$t('accessLogs.title')" icon="list" />

    <FilterCard :cols="3">
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('accessLogs.action') }}</span>
        <select
          v-model="action"
          class="field"
        >
          <option
            v-for="a in ACTIONS"
            :key="a || 'all'"
            :value="a"
          >
            {{ a ? a : $t('accessLogs.allActions') }}
          </option>
        </select>
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('accessLogs.ip') }}</span>
        <input
          v-model="ip"
          type="search"
          class="field"
          :placeholder="$t('accessLogs.ipPlaceholder')"
        >
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
    </FilterCard>

    <p
      v-if="error"
      class="mb-3 text-sm text-warn"
    >
      {{ error }}
    </p>

    <PageState :loading="loading && !items.length">
      <AccessLogTable :items="items" />
      <p
        v-if="items.length < total"
        class="mt-3"
      >
        <button
          type="button"
          class="btn"
          :disabled="loading"
          @click="load(false)"
        >
          {{ loading ? $t('common.loading') : $t('accessLogs.loadMore') }}
        </button>
      </p>
    </PageState>
  </div>
</template>
