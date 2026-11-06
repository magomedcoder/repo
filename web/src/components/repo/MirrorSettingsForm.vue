<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { reposApi } from '@/api'
import type { MirrorConfigureBody } from '@/api'
import type { MirrorConflictPolicy, MirrorDirection, MirrorDryRun, MirrorStatus, MirrorSyncEvent, Repository } from '@/api/types'
import { useAuth } from '@/composables/useAuth'
import { formatDate, i18n, localizeError } from '@/i18n'
import { formatBytes } from '@/lib/bytes'
import { hasMirror, isSSHURL, mirrorUiStatus, normalizeRefspecs, refspecsToText } from '@/lib/mirror'

const props = defineProps<{
  repoPath: string
  repo: Repository
}>()

const emit = defineEmits<{
  changed: []
}>()

const MIB = 1024 * 1024

const auth = useAuth()

const current = ref<Repository>(props.repo)
const configured = computed(() => hasMirror(current.value))

const enabled = ref(false)
const paused = ref(false)
const direction = ref<MirrorDirection>('pull')
const url = ref('')
const username = ref('')
const password = ref('')
const clearPassword = ref(false)
const hasPassword = ref(false)
const sshKey = ref('')
const sshPassphrase = ref('')
const clearSSHKey = ref(false)
const hasSSHKey = ref(false)
const interval = ref('')
const syncWindow = ref('')
const force = ref(true)
const prune = ref(false)
const refspecs = ref('')
const lfs = ref(false)
const partial = ref(false)
const verifySigned = ref(false)
const maxMiB = ref('')
const conflictPolicy = ref<MirrorConflictPolicy>('force')
const insecureTLS = ref(false)
const ownerOnly = ref(false)

const status = ref<MirrorStatus | null>(null)
const events = ref<MirrorSyncEvent[]>([])
const historyLoading = ref(false)
const historyError = ref('')
const dryRun = ref<MirrorDryRun | null>(null)
const dryRunBusy = ref(false)
const dryRunError = ref('')

const dryRunHasAction = computed(() => !!dryRun.value?.refs?.some((r) => r.action))

const busy = ref(false)
const checking = ref(false)
const error = ref('')
const message = ref('')
const checkMessage = ref('')
const checkOK = ref(false)

const inputCls = 'w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none'
const btnCls = 'inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-ink hover:bg-path disabled:cursor-not-allowed disabled:opacity-50'
const btnPrimaryCls = 'inline-flex items-center justify-center border border-ink bg-ink px-2.5 py-1 text-[10pt] text-paper hover:bg-ink-muted disabled:cursor-not-allowed disabled:opacity-50'

const canEditInsecureTLS = computed(() => auth.isAdmin.value || insecureTLS.value)
const showSSH = computed(() => isSSHURL(url.value) || hasSSHKey.value || !!sshKey.value)
const uiStatus = computed(() => mirrorUiStatus({
  mirror_direction: current.value.mirror_direction,
  mirror_enabled: paused.value ? false : current.value.mirror_enabled,
  mirror_status: status.value?.status ?? current.value.mirror_status,
  mirror_last_error: status.value?.last_error ?? current.value.mirror_last_error,
  mirror_last_sync_at: status.value?.last_sync_at ?? current.value.mirror_last_sync_at,
}))

const lastSyncAt = computed(() => status.value?.last_sync_at ?? current.value.mirror_last_sync_at ?? '')
const lastError = computed(() => status.value?.last_error ?? current.value.mirror_last_error ?? '')
const nextSyncAt = computed(() => status.value?.next_sync_at ?? current.value.mirror_next_sync_at ?? '')
const failCount = computed(() => status.value?.fail_count ?? current.value.mirror_fail_count ?? 0)
const remoteHead = computed(() => status.value?.remote_head ?? current.value.mirror_remote_head ?? '')
const lastProgress = computed(() => status.value?.last_progress ?? current.value.mirror_last_progress ?? '')

function syncFormFromRepo() {
  const r = current.value
  enabled.value = hasMirror(r)
  paused.value = hasMirror(r) && r.mirror_enabled === false
  direction.value = r.mirror_direction === 'push' ? 'push' : 'pull'
  url.value = r.mirror_url || ''
  username.value = r.mirror_username || ''
  password.value = ''
  clearPassword.value = false
  hasPassword.value = !!r.mirror_has_password
  sshKey.value = ''
  sshPassphrase.value = ''
  clearSSHKey.value = false
  hasSSHKey.value = !!r.mirror_has_ssh_key
  interval.value = r.mirror_interval || ''
  syncWindow.value = r.mirror_sync_window || ''
  force.value = r.mirror_force !== false
  prune.value = !!r.mirror_prune
  refspecs.value = refspecsToText(r.mirror_refspecs)
  lfs.value = !!r.mirror_lfs
  partial.value = !!r.mirror_partial
  verifySigned.value = !!r.mirror_verify_signed
  maxMiB.value = r.mirror_max_bytes ? String(Math.round((r.mirror_max_bytes / MIB) * 100) / 100) : ''
  conflictPolicy.value = r.mirror_conflict_policy === 'reject' ? 'reject' : 'force'
  insecureTLS.value = !!r.mirror_insecure_tls
  ownerOnly.value = !!r.mirror_owner_only
}

async function loadStatus() {
  if (!configured.value) {
    status.value = null
    return
  }

  try {
    status.value = await reposApi.mirrorStatus(props.repoPath)
  } catch {
    status.value = null
  }
}

async function loadHistory() {
  if (!configured.value) {
    events.value = []
    return
  }

  historyLoading.value = true
  historyError.value = ''
  try {
    const res = await reposApi.mirrorHistory(props.repoPath, { limit: 20 })
    events.value = res.events ?? []
  } catch (err) {
    historyError.value = localizeError(err, 'errors.loadMirrorHistory')
  } finally {
    historyLoading.value = false
  }
}

async function refreshAll() {
  await Promise.all([loadStatus(), loadHistory()])
}

async function refreshRepo() {
  emit('changed')
  try {
    current.value = await reposApi.get(props.repoPath)
    syncFormFromRepo()
  } catch {

  }

  await refreshAll()
}

async function refreshStatusOnly() {
  await Promise.all([loadStatus(), loadHistory()])
  try {
    const next = await reposApi.get(props.repoPath)
    current.value = { ...next }
  } catch {

  }
}

function buildBody(): MirrorConfigureBody {
  const body: MirrorConfigureBody = {
    enabled: !paused.value,
    direction: direction.value,
    url: url.value.trim(),
    username: username.value.trim(),
    interval: interval.value.trim(),
    sync_window: syncWindow.value.trim(),
    force: force.value,
    prune: prune.value,
    refspecs: direction.value === 'pull' ? normalizeRefspecs(refspecs.value) : '',
    lfs: lfs.value,
    partial: partial.value,
    verify_signed: verifySigned.value,
    max_bytes: Math.max(0, Math.round((Number(maxMiB.value) || 0) * MIB)),
    conflict_policy: conflictPolicy.value,
    owner_only: ownerOnly.value,
  }

  if (canEditInsecureTLS.value) {
    body.insecure_tls = insecureTLS.value
  }

  if (clearPassword.value) {
    body.clear_password = true
  } else if (password.value) {
    body.password = password.value
  }

  if (clearSSHKey.value) {
    body.clear_ssh_key = true
  } else if (sshKey.value.trim()) {
    body.ssh_private_key = sshKey.value.trim() + '\n'
    if (sshPassphrase.value) {
      body.ssh_passphrase = sshPassphrase.value
    }
  }

  return body
}

function resetFeedback() {
  error.value = ''
  message.value = ''
  checkMessage.value = ''
}

async function save() {
  busy.value = true
  resetFeedback()
  try {
    if (!enabled.value) {
      if (configured.value) {
        await clearMirror()
        return
      }

      return
    }

    await reposApi.configureMirror(props.repoPath, buildBody())
    message.value = i18n.global.t('settings.mirrorSaved')
    await refreshRepo()
  } catch (err) {
    error.value = localizeError(err, 'errors.saveFailed')
  } finally {
    busy.value = false
  }
}

async function clearMirror() {
  await reposApi.configureMirror(props.repoPath, { clear: true })
  message.value = i18n.global.t('settings.mirrorDisabled')
  events.value = []
  dryRun.value = null
  await refreshRepo()
}

async function removeMirror() {
  if (!confirm(i18n.global.t('mirrorForm.confirmRemove'))) {
    return
  }

  busy.value = true
  resetFeedback()
  try {
    await clearMirror()
  } catch (err) {
    error.value = localizeError(err, 'errors.saveFailed')
  } finally {
    busy.value = false
  }
}

async function syncNow() {
  busy.value = true
  resetFeedback()
  try {
    await reposApi.syncMirror(props.repoPath)
    message.value = i18n.global.t('settings.mirrorSynced')
  } catch (err) {
    error.value = localizeError(err, 'errors.mirrorSyncFailed')
  } finally {
    busy.value = false
    await refreshRepo()
  }
}

async function checkConnection() {
  checking.value = true
  resetFeedback()
  try {
    const res = await reposApi.checkMirror(props.repoPath)
    checkOK.value = res.ok !== false
    if (checkOK.value) {
      const head = res.head || res.default_branch
      const parts = [res.message || i18n.global.t('mirrorForm.checkOk')]
      if (head) {
        parts.push(i18n.global.t('mirror.remoteHead', { ref: head }))
      }

      if (res.refs) {
        parts.push(i18n.global.t('mirrorForm.refsFound', { n: res.refs.length }))
      }

      checkMessage.value = parts.join(' ')
    } else {
      checkMessage.value = res.message || i18n.global.t('mirrorForm.checkFailed')
    }
  } catch (err) {
    checkOK.value = false
    checkMessage.value = localizeError(err, 'errors.mirrorCheckFailed')
  } finally {
    checking.value = false
  }
}

async function runDryRun() {
  dryRunBusy.value = true
  dryRunError.value = ''
  dryRun.value = null
  try {
    dryRun.value = await reposApi.mirrorDryRun(props.repoPath)
  } catch (err) {
    dryRunError.value = localizeError(err, 'errors.mirrorDryRunFailed')
  } finally {
    dryRunBusy.value = false
  }
}

function triggerLabel(trigger: string) {
  const key = `mirror.trigger.${trigger}`
  return i18n.global.te(key) ? i18n.global.t(key) : trigger
}

function actionLabel(action: string) {
  const key = `mirror.action.${action}`
  return i18n.global.te(key) ? i18n.global.t(key) : action
}

function eventStatusLabel(status: string) {
  const key = `mirrorForm.event.${status}`
  return i18n.global.te(key) ? i18n.global.t(key) : status
}

function formatDuration(ms: number) {
  return ms < 1000 ? `${ms} ms` : `${(ms / 1000).toFixed(ms < 10_000 ? 1 : 0)} s`
}

function shortSha(sha?: string) {
  return sha ? sha.slice(0, 8) : '-'
}

let pollTimer: ReturnType<typeof setInterval> | undefined

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = undefined
  }
}

watch(
  () => uiStatus.value === 'syncing',
  (syncing) => {
    stopPolling()
    if (!syncing) {
      return
    }

    pollTimer = setInterval(() => {
      void refreshStatusOnly()
    }, 4000)
  },
  { immediate: true },
)

syncFormFromRepo()
void refreshAll()

onBeforeUnmount(stopPolling)
</script>

<template>
  <form class="overflow-hidden border border-line bg-paper" @submit.prevent="save">
    <div class="flex flex-wrap items-center justify-between gap-2 border-b border-line bg-path px-3 py-1.5 text-[10pt] text-ink">
      <span>{{ $t('settings.mirrorTitle') }}</span>
      <span v-if="uiStatus" class="text-[90%] text-ink-muted" :class="uiStatus === 'error' && 'text-warn'">
        {{ $t(`mirror.status.${uiStatus}`) }}
      </span>
    </div>
    <div class="space-y-4 p-3 sm:p-4">
      <p class="text-xs text-ink-muted">{{ $t('settings.mirrorIntro') }}</p>

      <label v-if="!configured" class="flex items-center gap-2 text-sm">
        <input v-model="enabled" type="checkbox" class="accent-ink">
        {{ $t('settings.mirrorEnable') }}
      </label>

      <template v-if="configured || enabled">
        <div v-if="configured" class="space-y-1 border-b border-line pb-3 text-xs text-ink-muted">
          <p v-if="lastSyncAt">{{ $t('settings.mirrorLastSync', { at: formatDate(lastSyncAt) }) }}</p>
          <p v-else>{{ $t('mirror.neverSynced') }}</p>
          <p v-if="nextSyncAt && uiStatus !== 'paused'">{{ $t('mirror.nextSync', { at: formatDate(nextSyncAt) }) }}</p>
          <p v-if="remoteHead">{{ $t('mirror.remoteHead', { ref: remoteHead }) }}</p>
          <p v-if="uiStatus === 'syncing' && lastProgress">{{ $t('mirror.progress', { text: lastProgress }) }}</p>
          <p v-if="failCount > 0" class="text-warn">{{ $t('mirror.failCount', { n: failCount }) }}</p>
          <p v-if="lastError" class="text-sm text-warn">{{ lastError }}</p>
        </div>

        <label v-if="configured" class="flex items-center gap-2 text-sm">
          <input v-model="paused" type="checkbox" class="accent-ink">
          {{ $t('mirrorForm.paused') }}
        </label>
        <p v-if="configured" class="-mt-2 text-xs text-ink-muted">{{ $t('mirrorForm.pausedHint') }}</p>

        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('settings.mirrorDirection') }}</span>
          <select v-model="direction" :class="inputCls">
            <option value="pull">{{ $t('settings.mirrorPull') }}</option>
            <option value="push">{{ $t('settings.mirrorPush') }}</option>
          </select>
        </label>
        <p class="text-xs text-ink-muted">
          {{ direction === 'push' ? $t('settings.mirrorPushHint') : $t('settings.mirrorPullHint') }}
        </p>

        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('settings.mirrorURL') }}</span>
          <input
            v-model="url"
            type="text"
            required
            spellcheck="false"
            autocapitalize="off"
            :class="[inputCls, 'font-mono']"
            :placeholder="$t('modal.mirrorURLPlaceholder')"
          >
        </label>

        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('settings.mirrorUsername') }}</span>
          <input v-model="username" :class="inputCls" :placeholder="$t('modal.optional')" autocomplete="off">
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('settings.mirrorPassword') }}</span>
          <input
            v-model="password"
            type="password"
            :class="inputCls"
            :placeholder="hasPassword ? $t('settings.mirrorPasswordKeep') : $t('modal.optional')"
            autocomplete="new-password"
          >
        </label>
        <label v-if="hasPassword" class="flex items-center gap-2 text-sm">
          <input v-model="clearPassword" type="checkbox" class="accent-ink">
          {{ $t('settings.mirrorClearPassword') }}
        </label>

        <template v-if="showSSH">
          <label class="block text-sm">
            <span class="mb-1 block font-semibold">{{ $t('mirrorForm.sshKey') }}</span>
            <textarea
              v-model="sshKey"
              rows="5"
              spellcheck="false"
              autocomplete="off"
              :class="[inputCls, 'resize-y font-mono']"
              :placeholder="hasSSHKey ? $t('mirrorForm.sshKeyKeep') : '-----BEGIN OPENSSH PRIVATE KEY-----'"
            />
          </label>
          <label class="block text-sm">
            <span class="mb-1 block font-semibold">{{ $t('mirrorForm.sshPassphrase') }}</span>
            <input v-model="sshPassphrase" type="password" :class="inputCls" :placeholder="$t('modal.optional')" autocomplete="new-password">
          </label>
          <label v-if="hasSSHKey" class="flex items-center gap-2 text-sm">
            <input v-model="clearSSHKey" type="checkbox" class="accent-ink">
            {{ $t('mirrorForm.clearSshKey') }}
          </label>
          <p class="text-xs text-ink-muted">{{ $t('mirrorForm.sshKeyHint') }}</p>
        </template>

        <div class="grid gap-4 sm:grid-cols-2">
          <label class="block text-sm">
            <span class="mb-1 block font-semibold">{{ $t('mirrorForm.interval') }}</span>
            <input v-model="interval" :class="[inputCls, 'font-mono']" :placeholder="$t('mirrorForm.intervalPlaceholder')" autocomplete="off">
          </label>
          <label class="block text-sm">
            <span class="mb-1 block font-semibold">{{ $t('mirrorForm.syncWindow') }}</span>
            <input v-model="syncWindow" :class="[inputCls, 'font-mono']" :placeholder="$t('mirrorForm.syncWindowPlaceholder')" autocomplete="off">
          </label>
        </div>
        <p class="-mt-2 text-xs text-ink-muted">{{ $t('mirrorForm.scheduleHint') }}</p>

        <label class="flex items-center gap-2 text-sm">
          <input v-model="force" type="checkbox" class="accent-ink">
          {{ direction === 'push' ? $t('mirrorForm.forcePush') : $t('mirrorForm.forcePull') }}
        </label>
        <label class="flex items-center gap-2 text-sm">
          <input v-model="prune" type="checkbox" class="accent-ink">
          {{ direction === 'push' ? $t('mirrorForm.prunePush') : $t('mirrorForm.prunePull') }}
        </label>
        <p v-if="force || prune" class="-mt-2 text-xs text-ink-muted">{{ $t('mirrorForm.forcePruneHint') }}</p>

        <label v-if="direction === 'pull'" class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('mirrorForm.refspecs') }}</span>
          <textarea
            v-model="refspecs"
            rows="3"
            spellcheck="false"
            :class="[inputCls, 'resize-y font-mono']"
            placeholder="+refs/heads/*:refs/heads/*&#10;+refs/tags/*:refs/tags/*"
          />
          <span class="mt-1 block text-xs text-ink-muted">{{ $t('mirrorForm.refspecsHint') }}</span>
        </label>

        <label v-if="direction === 'push'" class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('mirrorForm.conflictPolicy') }}</span>
          <select v-model="conflictPolicy" :class="inputCls">
            <option value="force">{{ $t('mirrorForm.conflictForce') }}</option>
            <option value="reject">{{ $t('mirrorForm.conflictReject') }}</option>
          </select>
          <span class="mt-1 block text-xs text-ink-muted">{{ $t('mirrorForm.conflictHint') }}</span>
        </label>

        <label class="flex items-center gap-2 text-sm">
          <input v-model="lfs" type="checkbox" class="accent-ink">
          {{ $t('mirrorForm.lfs') }}
        </label>

        <label class="flex items-center gap-2 text-sm">
          <input v-model="partial" type="checkbox" class="accent-ink">
          {{ $t('mirrorForm.partial') }}
        </label>
        <p class="-mt-2 text-xs text-ink-muted">{{ $t('mirrorForm.partialHint') }}</p>

        <label class="flex items-center gap-2 text-sm">
          <input v-model="verifySigned" type="checkbox" class="accent-ink">
          {{ $t('mirrorForm.verifySigned') }}
        </label>
        <p class="-mt-2 text-xs text-ink-muted">{{ $t('mirrorForm.verifySignedHint') }}</p>

        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('mirrorForm.maxMiB') }}</span>
          <input v-model="maxMiB" type="number" min="0" step="any" :class="inputCls" :placeholder="$t('mirrorForm.maxMiBPlaceholder')">
          <span class="mt-1 block text-xs text-ink-muted">
            {{ $t('mirrorForm.maxMiBHint') }}
            <template v-if="current.mirror_max_bytes">({{ formatBytes(current.mirror_max_bytes) }})</template>
          </span>
        </label>

        <template v-if="canEditInsecureTLS">
          <label class="flex items-center gap-2 text-sm">
            <input v-model="insecureTLS" type="checkbox" class="accent-ink">
            {{ $t('mirrorForm.insecureTls') }}
          </label>
          <p v-if="insecureTLS" class="border border-line bg-warn-soft px-2 py-1.5 text-xs text-warn">
            {{ $t('mirrorForm.insecureTlsWarn') }}
          </p>
        </template>

        <label class="flex items-center gap-2 text-sm">
          <input v-model="ownerOnly" type="checkbox" class="accent-ink">
          {{ $t('mirrorForm.ownerOnly') }}
        </label>
      </template>

      <p v-if="error" class="text-sm text-warn">{{ error }}</p>
      <p v-else-if="message" class="text-sm text-ink">{{ message }}</p>
      <p v-if="checkMessage" class="text-sm" :class="checkOK ? 'text-ink' : 'text-warn'">{{ checkMessage }}</p>

      <div class="flex flex-wrap gap-2">
        <button type="submit" :class="btnPrimaryCls" :disabled="busy">
          {{ busy ? $t('settings.saving') : $t('settings.mirrorSave') }}
        </button>
        <template v-if="configured">
          <button type="button" :class="btnCls" :disabled="busy || checking" @click="checkConnection">
            {{ checking ? $t('mirrorForm.checking') : $t('mirrorForm.check') }}
          </button>
          <button type="button" :class="btnCls" :disabled="busy || paused || uiStatus === 'syncing'" @click="syncNow">
            {{ $t('settings.mirrorSync') }}
          </button>
          <button
            type="button"
            class="inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-warn hover:bg-warn-soft disabled:cursor-not-allowed disabled:opacity-50"
            :disabled="busy"
            @click="removeMirror"
          >
            {{ $t('mirrorForm.remove') }}
          </button>
        </template>
      </div>
    </div>

    <template v-if="configured">
      <div class="flex items-center justify-between gap-2 border-y border-line bg-path px-3 py-1.5 text-[10pt] text-ink">
        <span>{{ $t('mirrorForm.historyTitle') }}</span>
        <button type="button" :class="btnCls" :disabled="historyLoading" @click="loadHistory">
          {{ historyLoading ? $t('common.loading') : $t('mirrorForm.refresh') }}
        </button>
      </div>
      <p v-if="historyError" class="px-3 pt-3 text-sm text-warn sm:px-4">{{ historyError }}</p>
      <div class="overflow-x-auto">
        <table class="w-full min-w-[640px] border-collapse text-left text-[10pt]">
          <thead class="bg-path text-ink">
            <tr>
              <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('mirrorForm.colStarted') }}</th>
              <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('mirrorForm.colFinished') }}</th>
              <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('mirrorForm.colTrigger') }}</th>
              <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('mirrorForm.colResult') }}</th>
              <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('mirrorForm.colDetails') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="ev in events" :key="ev.id" class="align-top odd:bg-paper even:bg-path/40">
              <td class="whitespace-nowrap border-b border-line px-2 py-1.5 text-ink-muted">{{ formatDate(ev.started_at) }}</td>
              <td class="whitespace-nowrap border-b border-line px-2 py-1.5 text-ink-muted">{{ ev.finished_at ? formatDate(ev.finished_at) : '-' }}</td>
              <td class="border-b border-line px-2 py-1.5 font-mono text-[90%]">{{ triggerLabel(ev.trigger) }}</td>
              <td class="border-b border-line px-2 py-1.5" :class="ev.status === 'failed' && 'text-warn'">
                {{ eventStatusLabel(ev.status) }}
              </td>
              <td class="break-all border-b border-line px-2 py-1.5 text-ink-muted" :class="ev.status === 'failed' && 'text-warn'">
                <template v-if="ev.error">{{ ev.error }}</template>
                <template v-else-if="ev.duration_ms != null">{{ $t('mirrorForm.duration', { n: formatDuration(ev.duration_ms) }) }}</template>
                <template v-else>-</template>
              </td>
            </tr>
            <tr v-if="!events.length">
              <td colspan="5" class="px-4 py-6 text-center text-sm text-ink-muted">{{ $t('mirrorForm.historyEmpty') }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <template v-if="direction === 'pull'">
        <div class="flex items-center justify-between gap-2 border-y border-line bg-path px-3 py-1.5 text-[10pt] text-ink">
          <span>{{ $t('mirrorForm.dryRunTitle') }}</span>
          <button type="button" :class="btnCls" :disabled="dryRunBusy" @click="runDryRun">
            {{ dryRunBusy ? $t('mirrorForm.dryRunning') : $t('mirrorForm.dryRunRun') }}
          </button>
        </div>
        <div class="p-3 sm:p-4">
          <p v-if="dryRunError" class="text-sm text-warn">{{ dryRunError }}</p>
          <p v-else-if="!dryRun && !dryRunBusy" class="text-xs text-ink-muted">{{ $t('mirrorForm.dryRunHint') }}</p>
          <template v-else-if="dryRun">
            <p v-if="dryRun.head || dryRun.default_branch" class="mb-2 text-xs text-ink-muted">{{ $t('mirror.remoteHead', { ref: dryRun.head || dryRun.default_branch }) }}</p>
            <p v-if="!dryRun.refs?.length" class="text-sm text-ink-muted">{{ $t('mirrorForm.dryRunEmpty') }}</p>
            <div v-else class="overflow-x-auto border border-line">
              <table class="w-full min-w-[520px] border-collapse text-left text-[10pt]">
                <thead class="bg-path text-ink">
                  <tr>
                    <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('mirrorForm.colRef') }}</th>
                    <th v-if="dryRunHasAction" class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('mirrorForm.colAction') }}</th>
                    <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('mirrorForm.colNew') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="r in dryRun.refs" :key="r.name" class="odd:bg-paper even:bg-path/40">
                    <td class="break-all border-b border-line px-2 py-1.5 font-mono text-[90%]">{{ r.name }}</td>
                    <td v-if="dryRunHasAction" class="border-b border-line px-2 py-1.5" :class="(r.action === 'delete' || r.action === 'forced') && 'text-warn'">
                      {{ r.action ? actionLabel(r.action) : '-' }}
                    </td>
                    <td class="border-b border-line px-2 py-1.5 font-mono text-[90%] text-ink-muted">{{ shortSha(r.commit_sha) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <p v-if="dryRun.truncated" class="mt-2 text-xs text-ink-muted">{{ $t('mirrorForm.dryRunTruncated') }}</p>
          </template>
        </div>
      </template>
    </template>
  </form>
</template>
