<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { adminAccessLogsApi } from '@/api'
import type { AccessLog } from '@/api/types'
import PageState from '@/components/ui/PageState.vue'
import { formatDate, localizeError } from '@/i18n'

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
    <div class="mb-4 border-b border-line pb-3">
      <h1 class="text-[140%] font-bold text-ink">{{ $t('accessLogs.title') }}</h1>
    </div>

    <div class="mb-3 grid gap-2 sm:grid-cols-3">
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('accessLogs.action') }}</span>
        <select
          v-model="action"
          class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none"
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
          class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none"
          :placeholder="$t('accessLogs.ipPlaceholder')"
        >
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('accessLogs.username') }}</span>
        <input
          v-model="username"
          type="search"
          class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none"
          :placeholder="$t('accessLogs.usernamePlaceholder')"
        >
      </label>
    </div>

    <p
      v-if="error"
      class="mb-3 text-sm text-warn"
    >
      {{ error }}
    </p>

    <PageState :loading="loading && !items.length">
      <div class="overflow-x-auto border border-line">
        <table class="w-full min-w-[760px] border-collapse text-left text-[10pt]">
          <thead class="bg-path text-ink">
            <tr>
              <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('accessLogs.time') }}</th>
              <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('accessLogs.action') }}</th>
              <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('accessLogs.user') }}</th>
              <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('accessLogs.ip') }}</th>
              <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('accessLogs.userAgent') }}</th>
              <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('accessLogs.details') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="row in items"
              :key="row.id"
              class="align-top odd:bg-paper even:bg-path/40"
            >
              <td class="border-b border-line px-2 py-1.5 whitespace-nowrap text-ink-muted">
                {{ formatDate(row.created_at) }}
              </td>
              <td class="border-b border-line px-2 py-1.5 font-mono text-[90%]">
                {{ row.action }}
              </td>
              <td class="border-b border-line px-2 py-1.5">
                {{ row.username || '-' }}
              </td>
              <td class="border-b border-line px-2 py-1.5 font-mono text-[90%]">
                {{ row.ip || '-' }}
              </td>
              <td
                class="border-b border-line px-2 py-1.5 max-w-[18rem] break-all font-mono text-[85%] text-ink-muted"
                :title="row.user_agent || undefined"
              >
                {{ row.user_agent || '-' }}
              </td>
              <td class="border-b border-line px-2 py-1.5 text-ink-muted">
                <span v-if="row.repo">{{ row.repo }}</span>
                <span v-if="row.ref"> @ {{ row.ref }}</span>
                <span v-if="row.path">{{ row.path }}</span>
                <span v-if="!row.repo && !row.ref && !row.path">-</span>
              </td>
            </tr>
            <tr v-if="!items.length">
              <td
                colspan="6"
                class="px-4 py-10 text-center text-sm text-ink-muted"
              >
                {{ $t('accessLogs.empty') }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p
        v-if="items.length < total"
        class="mt-3"
      >
        <button
          type="button"
          class="inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-ink hover:bg-path disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="loading"
          @click="load(false)"
        >
          {{ loading ? $t('common.loading') : $t('accessLogs.loadMore') }}
        </button>
      </p>
    </PageState>
  </div>
</template>
