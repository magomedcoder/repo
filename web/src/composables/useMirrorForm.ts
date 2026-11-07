import { computed, onBeforeUnmount, ref, watch, type Ref } from 'vue'
import { reposApi } from '@/api'
import type { MirrorConfigureBody } from '@/api'
import type { MirrorConflictPolicy, MirrorDirection, MirrorDryRun, MirrorStatus, MirrorSyncEvent, Repository } from '@/api/types'
import { useAuth } from '@/composables/useAuth'
import { i18n, localizeError } from '@/i18n'
import { hasMirror, isSSHURL, mirrorUiStatus, normalizeRefspecs, refspecsToText } from '@/lib/mirror'

const MIB = 1024 * 1024

export const mirrorInputCls = 'field'
export const mirrorBtnCls = 'btn'
export const mirrorBtnPrimaryCls = 'btn-primary'

export function useMirrorForm(
  repoPath: Ref<string> | (() => string),
  repo: Ref<Repository> | (() => Repository),
  emitChanged: () => void,
) {
  const auth = useAuth()

  const path = () => (typeof repoPath === 'function' ? repoPath() : repoPath.value)
  const repoSnapshot = () => (typeof repo === 'function' ? repo() : repo.value)

  const current = ref<Repository>(repoSnapshot())
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

  const canEditInsecureTLS = computed(() => auth.isAdmin.value || insecureTLS.value)
  const showSSH = computed(() => isSSHURL(url.value) || hasSSHKey.value || !!sshKey.value)
  const uiStatus = computed(() =>
    mirrorUiStatus({
      mirror_direction: current.value.mirror_direction,
      mirror_enabled: paused.value ? false : current.value.mirror_enabled,
      mirror_status: status.value?.status ?? current.value.mirror_status,
      mirror_last_error: status.value?.last_error ?? current.value.mirror_last_error,
      mirror_last_sync_at: status.value?.last_sync_at ?? current.value.mirror_last_sync_at,
    }),
  )

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
      status.value = await reposApi.mirrorStatus(path())
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
      const res = await reposApi.mirrorHistory(path(), { limit: 20 })
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
    emitChanged()
    try {
      current.value = await reposApi.get(path())
      syncFormFromRepo()
    } catch {

    }

    await refreshAll()
  }

  async function refreshStatusOnly() {
    await Promise.all([loadStatus(), loadHistory()])
    try {
      const next = await reposApi.get(path())
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

      await reposApi.configureMirror(path(), buildBody())
      message.value = i18n.global.t('settings.mirrorSaved')
      await refreshRepo()
    } catch (err) {
      error.value = localizeError(err, 'errors.saveFailed')
    } finally {
      busy.value = false
    }
  }

  async function clearMirror() {
    await reposApi.configureMirror(path(), { clear: true })
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
      await reposApi.syncMirror(path())
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
      const res = await reposApi.checkMirror(path())
      checkOK.value = res.ok !== false
      if (checkOK.value) {
        const head = res.head || res.default_branch
        const parts = [res.message || i18n.global.t('mirrorForm.checkOk')]
        if (head) {
          parts.push(i18n.global.t('mirror.remoteHead', { 
            ref: head 
          }))
        }

        if (res.refs) {
          parts.push(i18n.global.t('mirrorForm.refsFound', {
            n: res.refs.length
          }))
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
      dryRun.value = await reposApi.mirrorDryRun(path())
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

  function eventStatusLabel(statusKey: string) {
    const key = `mirrorForm.event.${statusKey}`
    return i18n.global.te(key) ? i18n.global.t(key) : statusKey
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

  return {
    current,
    configured,
    enabled,
    paused,
    direction,
    url,
    username,
    password,
    clearPassword,
    hasPassword,
    sshKey,
    sshPassphrase,
    clearSSHKey,
    hasSSHKey,
    interval,
    syncWindow,
    force,
    prune,
    refspecs,
    lfs,
    partial,
    verifySigned,
    maxMiB,
    conflictPolicy,
    insecureTLS,
    ownerOnly,
    status,
    events,
    historyLoading,
    historyError,
    dryRun,
    dryRunBusy,
    dryRunError,
    dryRunHasAction,
    busy,
    checking,
    error,
    message,
    checkMessage,
    checkOK,
    canEditInsecureTLS,
    showSSH,
    uiStatus,
    lastSyncAt,
    lastError,
    nextSyncAt,
    failCount,
    remoteHead,
    lastProgress,
    inputCls: mirrorInputCls,
    btnCls: mirrorBtnCls,
    btnPrimaryCls: mirrorBtnPrimaryCls,
    loadHistory,
    save,
    removeMirror,
    syncNow,
    checkConnection,
    runDryRun,
    triggerLabel,
    actionLabel,
    eventStatusLabel,
    formatDuration,
    shortSha,
  }
}

export type UseMirrorFormReturn = ReturnType<typeof useMirrorForm>
