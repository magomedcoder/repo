<script setup lang="ts">
import { computed } from 'vue'
import MirrorConfigFields from '@/components/repo/mirror/MirrorConfigFields.vue'
import MirrorHistoryPanel from '@/components/repo/mirror/MirrorHistoryPanel.vue'
import MirrorStatusPanel from '@/components/repo/mirror/MirrorStatusPanel.vue'
import type { Repository } from '@/api/types'
import { useMirrorForm } from '@/composables/useMirrorForm'

const props = defineProps<{
  repoPath: string
  repo: Repository
}>()

const emit = defineEmits<{
  changed: []
}>()

const {
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
  inputCls,
  btnCls,
  btnPrimaryCls,
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
} = useMirrorForm(
  () => props.repoPath,
  () => props.repo,
  () => emit('changed'),
)

const statusBindings = computed(() => ({
  configured: configured.value,
  uiStatus: uiStatus.value,
  lastSyncAt: lastSyncAt.value,
  nextSyncAt: nextSyncAt.value,
  remoteHead: remoteHead.value,
  lastProgress: lastProgress.value,
  failCount: failCount.value,
  lastError: lastError.value,
  error: error.value,
  message: message.value,
  checkMessage: checkMessage.value,
  checkOK: checkOK.value,
  busy: busy.value,
  checking: checking.value,
  paused: paused.value,
  btnCls,
  btnPrimaryCls,
}))
</script>

<template>
  <form class="surface" @submit.prevent="save">
    <div class="surface-header flex-wrap justify-between">
      <span>{{ $t('settings.mirrorTitle') }}</span>
      <span v-if="uiStatus" class="text-[90%] text-ink-muted" :class="uiStatus === 'error' && 'text-warn'">
        {{ $t(`mirror.status.${uiStatus}`) }}
      </span>
    </div>
    <div class="space-y-4 p-3 sm:p-4">
      <p class="text-xs text-ink-muted">{{ $t('settings.mirrorIntro') }}</p>

      <MirrorConfigFields
        v-model:enabled="enabled"
        v-model:paused="paused"
        v-model:direction="direction"
        v-model:url="url"
        v-model:username="username"
        v-model:password="password"
        v-model:clear-password="clearPassword"
        v-model:ssh-key="sshKey"
        v-model:ssh-passphrase="sshPassphrase"
        v-model:clearSSHKey="clearSSHKey"
        v-model:interval="interval"
        v-model:sync-window="syncWindow"
        v-model:force="force"
        v-model:prune="prune"
        v-model:refspecs="refspecs"
        v-model:lfs="lfs"
        v-model:partial="partial"
        v-model:verify-signed="verifySigned"
        v-model:maxMiB="maxMiB"
        v-model:conflict-policy="conflictPolicy"
        v-model:insecureTLS="insecureTLS"
        v-model:owner-only="ownerOnly"
        :configured="configured"
        :has-password="hasPassword"
        :hasSSHKey="hasSSHKey"
        :showSSH="showSSH"
        :canEditInsecureTLS="canEditInsecureTLS"
        :current="current"
        :input-cls="inputCls"
      >
        <template v-if="configured" #before-fields>
          <MirrorStatusPanel
            v-bind="statusBindings"
            section="summary"
            @check="checkConnection"
            @sync="syncNow"
            @remove="removeMirror"
          />
        </template>
      </MirrorConfigFields>

      <MirrorStatusPanel
        v-if="configured || enabled"
        v-bind="statusBindings"
        section="actions"
        @check="checkConnection"
        @sync="syncNow"
        @remove="removeMirror"
      />
    </div>

    <MirrorHistoryPanel
      :configured="configured"
      :direction="direction"
      :events="events"
      :history-loading="historyLoading"
      :history-error="historyError"
      :dry-run="dryRun"
      :dry-run-busy="dryRunBusy"
      :dry-run-error="dryRunError"
      :dry-run-has-action="dryRunHasAction"
      :btn-cls="btnCls"
      :trigger-label="triggerLabel"
      :event-status-label="eventStatusLabel"
      :format-duration="formatDuration"
      :action-label="actionLabel"
      :short-sha="shortSha"
      @refresh-history="loadHistory"
      @dry-run="runDryRun"
    />
  </form>
</template>
