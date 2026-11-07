<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { adminMirrorsApi } from '@/api'
import type { AdminMirror } from '@/api/types'
import MirrorAdminTable from '@/components/admin/MirrorAdminTable.vue'
import PageState from '@/components/ui/PageState.vue'
import { i18n, localizeError } from '@/i18n'
import Icon from '@/components/ui/Icon.vue'
import PageHeader from '@/components/ui/PageHeader.vue'
import FilterCard from '@/components/layout/FilterCard.vue'

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

const btnCls = 'btn'

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
    <PageHeader :title="$t('adminMirrors.title')" icon="mirror">
      <template #actions>
        <button type="button" :class="btnCls" :disabled="busy || loading" @click="syncAll">
          <Icon name="sync" size="sm" />
          {{ $t('adminMirrors.syncAll') }}
        </button>
      </template>
    </PageHeader>

    <FilterCard :cols="3">
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('adminMirrors.direction') }}</span>
        <select v-model="direction" class="field">
          <option v-for="d in DIRECTIONS" :key="d || 'all'" :value="d">
            {{ d === '' ? $t('adminMirrors.allDirections') : d === 'pull' ? $t('settings.mirrorPull') : $t('settings.mirrorPush') }}
          </option>
        </select>
      </label>
      <label class="flex items-end gap-2 pb-1 text-sm">
        <input v-model="onlyErrors" type="checkbox" class="accent-ink">
        {{ $t('adminMirrors.onlyErrors') }}
      </label>
    </FilterCard>

    <div class="mb-3 flex flex-wrap items-center gap-2">
      <button type="button" :class="btnCls" :disabled="busy || !selected.length" @click="syncSelected">
        <Icon name="sync" size="sm" />
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
      <MirrorAdminTable
        :items="items"
        :selected="selected"
        :busy="busy"
        :btn-class="btnCls"
        @update:selected="selected = $event"
        @toggle-all="toggleAll"
        @sync="syncOne"
        @pause="pauseOne"
      />
    </PageState>
  </div>
</template>
