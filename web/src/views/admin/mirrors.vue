<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { adminMirrorsApi } from '@/api'
import type { AdminMirror, MirrorGroup } from '@/api/types'
import type { MirrorHealth } from '@/api'
import MirrorAdminTable from '@/components/admin/MirrorAdminTable.vue'
import CpPage from '@/components/admin/CpPage.vue'
import FilterCard from '@/components/layout/FilterCard.vue'
import PageState from '@/components/ui/PageState.vue'
import Icon from '@/components/ui/Icon.vue'
import Surface from '@/components/ui/Surface.vue'
import { i18n, localizeError } from '@/i18n'

const DIRECTIONS = ['', 'pull', 'push'] as const
const route = useRoute()

const items = ref<AdminMirror[]>([])
const groups = ref<MirrorGroup[]>([])
const health = ref<MirrorHealth | null>(null)
const direction = ref<(typeof DIRECTIONS)[number]>('')
const onlyErrors = ref(route.query.errors === '1')
const q = ref('')
const groupId = ref<number | ''>('')
const selected = ref<number[]>([])
const loading = ref(true)
const busy = ref(false)
const error = ref('')
const message = ref('')
const showImport = ref(false)
const showGroupForm = ref(false)
const importUrls = ref('')
const importPrivate = ref(false)
const groupName = ref('')

const allSelected = computed(() => items.value.length > 0 && selected.value.length === items.value.length)
const anySyncing = computed(() => items.value.some((m) => m.status === 'syncing'))
const btnCls = 'btn'

async function load(silent = false) {
  if (!silent) {
    loading.value = true
  }

  error.value = ''
  try {
    const [res, groupsRes, healthRes] = await Promise.all([
      adminMirrorsApi.list({
        direction: direction.value || undefined,
        error: onlyErrors.value || undefined,
        q: q.value.trim() || undefined,
        group_id: groupId.value === '' ? undefined : groupId.value,
      }),
      adminMirrorsApi.listGroups().catch(() => ({ groups: [] as MirrorGroup[] })),
      adminMirrorsApi.health(3600).catch(() => null),
    ])
    items.value = res.mirrors ?? []
    groups.value = groupsRes.groups ?? []
    health.value = healthRes
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
  if (!confirm(i18n.global.t('adminMirrors.confirmSyncAll'))) return
  return run(() => adminMirrorsApi.sync(), 'adminMirrors.syncAllQueued')
}

function setPaused(paused: boolean) {
  const ids = [...selected.value]
  return run(
    () => adminMirrorsApi.pause(ids, paused),
    paused ? 'adminMirrors.paused' : 'adminMirrors.resumed',
    ids.length,
  )
}

function pauseOne(m: AdminMirror) {
  return run(
    () => adminMirrorsApi.pause([m.id], !m.paused),
    m.paused ? 'adminMirrors.resumed' : 'adminMirrors.paused',
    1,
  )
}

function syncOne(m: AdminMirror) {
  return run(() => adminMirrorsApi.sync([m.id]), 'adminMirrors.syncQueued', 1)
}

async function verifyAll() {
  busy.value = true
  error.value = ''
  message.value = ''
  try {
    const res = await adminMirrorsApi.verify({
      direction: direction.value || undefined,
      q: q.value.trim() || undefined,
    })
    message.value = i18n.global.t('cp.mirrors.verifyDone', {
      checked: res.checked,
      failed: res.failed,
    })
  } catch (err) {
    error.value = localizeError(err, 'errors.mirrorBulkFailed')
  } finally {
    busy.value = false
  }
}

async function submitImport() {
  const urls = importUrls.value
    .split(/\n+/)
    .map((s) => s.trim())
    .filter(Boolean)
  if (!urls.length) {
    return
  }

  busy.value = true
  error.value = ''
  message.value = ''
  try {
    const res = await adminMirrorsApi.importOrg({
      urls,
      private: importPrivate.value,
      group_id: groupId.value === '' ? undefined : groupId.value,
    })
    const created = res.results?.filter((r) => r.created).length ?? 0
    const failed = res.results?.filter((r) => r.error).length ?? 0
    message.value = i18n.global.t('cp.mirrors.importDone', { created, failed })
    showImport.value = false
    importUrls.value = ''
    await load(true)
  } catch (err) {
    error.value = localizeError(err, 'errors.mirrorBulkFailed')
  } finally {
    busy.value = false
  }
}

async function createGroup() {
  const name = groupName.value.trim()
  if (!name) {
    return
  }

  busy.value = true
  error.value = ''
  try {
    await adminMirrorsApi.createGroup({ name })
    groupName.value = ''
    showGroupForm.value = false
    await load(true)
  } catch (err) {
    error.value = localizeError(err, 'errors.mirrorBulkFailed')
  } finally {
    busy.value = false
  }
}

async function removeGroup(id: number) {
  if (!confirm(i18n.global.t('cp.mirrors.confirmDeleteGroup'))) {
    return
  }
  
  busy.value = true
  try {
    await adminMirrorsApi.deleteGroup(id)
    if (groupId.value === id) groupId.value = ''
    await load(true)
  } catch (err) {
    error.value = localizeError(err, 'errors.deleteFailed')
  } finally {
    busy.value = false
  }
}

watch([direction, onlyErrors, groupId], () => {
  void load()
})

let searchTimer: ReturnType<typeof setTimeout> | undefined
watch(q, () => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    void load()
  }, 250)
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
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<template>
  <CpPage :title="$t('cp.mirrors.title')" :description="$t('cp.mirrors.description')">
    <template #actions>
      <button type="button" :class="btnCls" :disabled="busy || loading" @click="verifyAll">
        <Icon name="check" size="sm" />
        {{ $t('cp.mirrors.verify') }}
      </button>
      <button type="button" :class="btnCls" :disabled="busy || loading" @click="showImport = !showImport">
        <Icon name="download" size="sm" />
        {{ $t('cp.mirrors.import') }}
      </button>
      <button type="button" :class="btnCls" :disabled="busy || loading" @click="syncAll">
        <Icon name="sync" size="sm" />
        {{ $t('adminMirrors.syncAll') }}
      </button>
    </template>

    <div
      v-if="health"
      class="mb-3 flex flex-wrap items-center gap-2 rounded-md border px-3 py-2 text-[12px]"
      :class="health.healthy ? 'border-line-strong/50 bg-white/40 text-ink-muted' : 'border-warn/40 bg-warn/10 text-warn'"
    >
      <Icon :name="health.healthy ? 'check' : 'alert'" size="sm" />
      <span v-if="health.healthy">{{ $t('cp.mirrors.healthOk') }}</span>
      <span v-else>{{ $t('cp.mirrors.healthBad', { n: health.in_error }) }}</span>
    </div>

    <FilterCard :cols="3">
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('adminMirrors.direction') }}</span>
        <select v-model="direction" class="field">
          <option v-for="d in DIRECTIONS" :key="d || 'all'" :value="d">
            {{ d === '' ? $t('adminMirrors.allDirections') : d === 'pull' ? $t('settings.mirrorPull') : $t('settings.mirrorPush') }}
          </option>
        </select>
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('cp.mirrors.group') }}</span>
        <select v-model="groupId" class="field">
          <option value="">{{ $t('cp.mirrors.allGroups') }}</option>
          <option v-for="g in groups" :key="g.id" :value="g.id">{{ g.name }}</option>
        </select>
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('cp.mirrors.search') }}</span>
        <input v-model="q" type="search" class="field" :placeholder="$t('cp.mirrors.search')">
      </label>
      <label class="flex items-end gap-2 pb-1 text-sm">
        <input v-model="onlyErrors" type="checkbox" class="accent-ink">
        {{ $t('adminMirrors.onlyErrors') }}
      </label>
    </FilterCard>

    <Surface v-if="showImport" as="form" class="mb-3" @submit.prevent="submitImport">
      <div class="border-b border-line-strong/40 px-3 py-2 text-[13px] font-semibold sm:px-4">
        {{ $t('cp.mirrors.import') }}
      </div>
      <div class="space-y-3 p-3 sm:p-4">
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('cp.mirrors.importUrls') }}</span>
          <textarea v-model="importUrls" class="field min-h-28 font-mono text-[12px]" required />
        </label>
        <label class="flex items-center gap-2 text-sm">
          <input v-model="importPrivate" type="checkbox" class="accent-ink">
          {{ $t('settings.private') }}
        </label>
        <button type="submit" class="btn-primary" :disabled="busy">{{ $t('cp.mirrors.importRun') }}</button>
      </div>
    </Surface>

    <Surface class="mb-3 overflow-hidden">
      <div class="flex flex-wrap items-center justify-between gap-2 border-b border-line-strong/40 px-3 py-2 sm:px-4">
        <h3 class="text-[13px] font-semibold">{{ $t('cp.mirrors.groups') }}</h3>
        <button type="button" class="btn" @click="showGroupForm = !showGroupForm">
          <Icon name="plus" size="sm" />
          {{ $t('cp.mirrors.addGroup') }}
        </button>
      </div>
      <form v-if="showGroupForm" class="flex flex-wrap gap-2 border-b border-line-strong/30 px-3 py-2 sm:px-4" @submit.prevent="createGroup">
        <input v-model="groupName" class="field min-w-[12rem] flex-1" :placeholder="$t('cp.mirrors.groupName')" required>
        <button type="submit" class="btn-primary" :disabled="busy">{{ $t('common.ok') }}</button>
      </form>
      <ul v-if="groups.length" class="divide-y divide-line-strong/30">
        <li
          v-for="g in groups"
          :key="g.id"
          class="flex items-center justify-between gap-2 px-3 py-2 text-[12px] sm:px-4"
        >
          <span class="font-medium text-ink">{{ g.name }}</span>
          <button type="button" class="btn-warn" :disabled="busy" @click="removeGroup(g.id)">
            <Icon name="trash" size="sm" />
          </button>
        </li>
      </ul>
      <p v-else class="px-3 py-3 text-[12px] text-ink-muted sm:px-4">{{ $t('cp.mirrors.noGroups') }}</p>
    </Surface>

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
  </CpPage>
</template>
