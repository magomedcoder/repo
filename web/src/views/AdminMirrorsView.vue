<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { adminMirrorsApi } from '@/api'
import type { AdminMirror } from '@/api/types'
import PageState from '@/components/ui/PageState.vue'
import { formatDate, i18n, localizeError } from '@/i18n'
import { mirrorRemoteLabel, mirrorUiStatus } from '@/lib/mirror'
import { repoLocation } from '@/lib/repoNav'

const DIRECTIONS = ['', 'pull', 'push'] as const

const items = ref<AdminMirror[]>([])
const direction = ref<(typeof DIRECTIONS)[number]>('')
const onlyErrors = ref(false)
const selected = ref<number[]>([])
const loading = ref(true)
const busy = ref(false)
const error = ref('')
const message = ref('')

const allSelected = computed(() => items.value.length > 0 && selected.value.length === items.value.length)
const anySyncing = computed(() => items.value.some((m) => m.status === 'syncing'))

const btnCls = 'inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-ink hover:bg-path disabled:cursor-not-allowed disabled:opacity-50'

function rowStatus(m: AdminMirror) {
  return mirrorUiStatus({
    mirror_direction: m.direction,
    mirror_enabled: m.paused ? false : undefined,
    mirror_status: m.status,
    mirror_last_error: m.last_error,
    mirror_last_sync_at: m.last_sync_at,
  }) || 'idle'
}

async function load(silent = false) {
  if (!silent) {
    loading.value = true
  }

  error.value = ''
  try {
    const res = await adminMirrorsApi.list({
      direction: direction.value || undefined,
      error: onlyErrors.value || undefined,
    })
    items.value = res.mirrors ?? []
    const ids = new Set(items.value.map((m) => m.id))
    selected.value = selected.value.filter((id) => ids.has(id))
  } catch (err) {
    error.value = localizeError(err, 'errors.loadMirrors')
  } finally {
    loading.value = false
  }
}

function toggleAll() {
  selected.value = allSelected.value ? [] : items.value.map((m) => m.id)
}

async function run(action: () => Promise<unknown>, doneKey: string, n?: number) {
  busy.value = true
  error.value = ''
  message.value = ''
  try {
    await action()
    message.value = i18n.global.t(doneKey, { n: n ?? 0 })
    await load(true)
  } catch (err) {
    error.value = localizeError(err, 'errors.mirrorBulkFailed')
  } finally {
    busy.value = false
  }
}

function syncSelected() {
  const ids = [...selected.value]
  return run(() => adminMirrorsApi.sync(ids), 'adminMirrors.syncQueued', ids.length)
}

function syncAll() {
  if (!confirm(i18n.global.t('adminMirrors.confirmSyncAll'))) {
    return
  }

  return run(() => adminMirrorsApi.sync(), 'adminMirrors.syncAllQueued')
}

function setPaused(paused: boolean) {
  const ids = [...selected.value]
  return run(() => adminMirrorsApi.pause(ids, paused), paused ? 'adminMirrors.paused' : 'adminMirrors.resumed', ids.length)
}

function pauseOne(m: AdminMirror) {
  return run(() => adminMirrorsApi.pause([m.id], !m.paused), m.paused ? 'adminMirrors.resumed' : 'adminMirrors.paused', 1)
}

function syncOne(m: AdminMirror) {
  return run(() => adminMirrorsApi.sync([m.id]), 'adminMirrors.syncQueued', 1)
}

watch([direction, onlyErrors], () => {
  void load()
})

let pollTimer: ReturnType<typeof setInterval> | undefined

watch(anySyncing, (syncing) => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = undefined
  }

  if (syncing) {
    pollTimer = setInterval(() => {
      void load(true)
    }, 5000)
  }
})

onMounted(() => {
  void load()
})

onBeforeUnmount(() => {
  if (pollTimer) {
    clearInterval(pollTimer)
  }
})
</script>

<template>
  <div class="mx-auto max-w-6xl">
    <div class="mb-4 flex flex-wrap items-center justify-between gap-2 border-b border-line pb-3">
      <h1 class="text-[140%] font-bold text-ink">{{ $t('adminMirrors.title') }}</h1>
      <button type="button" :class="btnCls" :disabled="busy || loading" @click="syncAll">
        {{ $t('adminMirrors.syncAll') }}
      </button>
    </div>

    <div class="mb-3 grid gap-2 sm:grid-cols-3">
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('adminMirrors.direction') }}</span>
        <select v-model="direction" class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none">
          <option v-for="d in DIRECTIONS" :key="d || 'all'" :value="d">
            {{ d === '' ? $t('adminMirrors.allDirections') : d === 'pull' ? $t('settings.mirrorPull') : $t('settings.mirrorPush') }}
          </option>
        </select>
      </label>
      <label class="flex items-end gap-2 pb-1 text-sm">
        <input v-model="onlyErrors" type="checkbox" class="accent-ink">
        {{ $t('adminMirrors.onlyErrors') }}
      </label>
    </div>

    <div class="mb-3 flex flex-wrap items-center gap-2">
      <button type="button" :class="btnCls" :disabled="busy || !selected.length" @click="syncSelected">
        {{ $t('adminMirrors.syncSelected') }}
      </button>
      <button type="button" :class="btnCls" :disabled="busy || !selected.length" @click="setPaused(true)">
        {{ $t('adminMirrors.pauseSelected') }}
      </button>
      <button type="button" :class="btnCls" :disabled="busy || !selected.length" @click="setPaused(false)">
        {{ $t('adminMirrors.resumeSelected') }}
      </button>
      <span v-if="selected.length" class="text-sm text-ink-muted">{{ $t('adminMirrors.selected', { n: selected.length }) }}</span>
    </div>

    <p v-if="error" class="mb-3 text-sm text-warn">{{ error }}</p>
    <p v-else-if="message" class="mb-3 text-sm text-ink">{{ message }}</p>

    <PageState :loading="loading && !items.length">
      <div class="overflow-x-auto border border-line">
        <table class="w-full min-w-[960px] border-collapse text-left text-[10pt]">
          <thead class="bg-path text-ink">
            <tr>
              <th class="w-8 border-b border-line px-2 py-1.5">
                <input
                  type="checkbox"
                  class="accent-ink"
                  :checked="allSelected"
                  :aria-label="$t('adminMirrors.selectAll')"
                  @change="toggleAll"
                >
              </th>
              <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('adminMirrors.repository') }}</th>
              <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('adminMirrors.direction') }}</th>
              <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('adminMirrors.status') }}</th>
              <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('adminMirrors.lastSync') }}</th>
              <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('adminMirrors.nextSync') }}</th>
              <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('adminMirrors.lastError') }}</th>
              <th class="border-b border-line px-2 py-1.5" />
            </tr>
          </thead>
          <tbody>
            <tr v-for="m in items" :key="m.id" class="align-top odd:bg-paper even:bg-path/40">
              <td class="border-b border-line px-2 py-1.5">
                <input v-model="selected" type="checkbox" class="accent-ink" :value="m.id" :aria-label="m.repo_path">
              </td>
              <td class="border-b border-line px-2 py-1.5">
                <RouterLink :to="repoLocation('repo-settings', m.repo_path)" class="font-mono text-[90%] no-underline hover:underline">
                  {{ m.repo_path }}
                </RouterLink>
                <p v-if="m.url" class="break-all font-mono text-[80%] text-ink-muted">{{ mirrorRemoteLabel(m.url) }}</p>
              </td>
              <td class="border-b border-line px-2 py-1.5 font-mono text-[90%]">{{ m.direction }}</td>
              <td class="whitespace-nowrap border-b border-line px-2 py-1.5" :class="rowStatus(m) === 'error' && 'text-warn'">
                {{ $t(`mirror.status.${rowStatus(m)}`) }}
                <span v-if="(m.fail_count ?? 0) > 0" class="text-ink-muted"> *{{ m.fail_count }}</span>
              </td>
              <td class="whitespace-nowrap border-b border-line px-2 py-1.5 text-ink-muted">
                {{ m.last_sync_at ? formatDate(m.last_sync_at) : '-' }}
              </td>
              <td class="whitespace-nowrap border-b border-line px-2 py-1.5 text-ink-muted">
                {{ m.next_sync_at && !m.paused ? formatDate(m.next_sync_at) : '-' }}
              </td>
              <td class="max-w-[22rem] break-words border-b border-line px-2 py-1.5 text-warn">
                {{ m.last_error || '' }}
              </td>
              <td class="whitespace-nowrap border-b border-line px-2 py-1.5 text-right">
                <button type="button" :class="[btnCls, 'mr-1 px-2 py-0.5']" :disabled="busy || m.paused" @click="syncOne(m)">
                  {{ $t('settings.mirrorSync') }}
                </button>
                <button type="button" :class="[btnCls, 'px-2 py-0.5']" :disabled="busy" @click="pauseOne(m)">
                  {{ m.paused ? $t('adminMirrors.resume') : $t('adminMirrors.pause') }}
                </button>
              </td>
            </tr>
            <tr v-if="!items.length">
              <td colspan="8" class="px-4 py-10 text-center text-sm text-ink-muted">{{ $t('adminMirrors.empty') }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </PageState>
  </div>
</template>
