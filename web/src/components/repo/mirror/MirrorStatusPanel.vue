<script setup lang="ts">
import { formatDate } from '@/i18n'
import type { MirrorUiStatus } from '@/lib/mirror'

const props = withDefaults(
  defineProps<{
    configured: boolean
    section?: 'summary' | 'actions' | 'all'
    uiStatus: MirrorUiStatus | ''
    lastSyncAt: string
    nextSyncAt: string
    remoteHead: string
    lastProgress: string
    failCount: number
    lastError: string
    error: string
    message: string
    checkMessage: string
    checkOK: boolean
    busy: boolean
    checking: boolean
    paused: boolean
    btnCls: string
    btnPrimaryCls: string
  }>(),
  { section: 'all' },
)

const emit = defineEmits<{
  check: []
  sync: []
  remove: []
}>()

const showSummary = () => props.section === 'summary' || props.section === 'all'
const showActions = () => props.section === 'actions' || props.section === 'all'
</script>

<template>
  <div v-if="configured && showSummary()" class="space-y-1 border-b border-line pb-3 text-xs text-ink-muted">
    <p v-if="lastSyncAt">{{ $t('settings.mirrorLastSync', { at: formatDate(lastSyncAt) }) }}</p>
    <p v-else>{{ $t('mirror.neverSynced') }}</p>
    <p v-if="nextSyncAt && uiStatus !== 'paused'">{{ $t('mirror.nextSync', { at: formatDate(nextSyncAt) }) }}</p>
    <p v-if="remoteHead">{{ $t('mirror.remoteHead', { ref: remoteHead }) }}</p>
    <p v-if="uiStatus === 'syncing' && lastProgress">{{ $t('mirror.progress', { text: lastProgress }) }}</p>
    <p v-if="failCount > 0" class="text-warn">{{ $t('mirror.failCount', { n: failCount }) }}</p>
    <p v-if="lastError" class="text-sm text-warn">{{ lastError }}</p>
  </div>

  <template v-if="showActions()">
  <p v-if="error" class="text-sm text-warn">{{ error }}</p>
  <p v-else-if="message" class="text-sm text-ink">{{ message }}</p>
  <p v-if="checkMessage" class="text-sm" :class="checkOK ? 'text-ink' : 'text-warn'">{{ checkMessage }}</p>

  <div class="flex flex-wrap gap-2">
    <button type="submit" :class="btnPrimaryCls" :disabled="busy">
      {{ busy ? $t('settings.saving') : $t('settings.mirrorSave') }}
    </button>
    <template v-if="configured">
      <button type="button" :class="btnCls" :disabled="busy || checking" @click="emit('check')">
        {{ checking ? $t('mirrorForm.checking') : $t('mirrorForm.check') }}
      </button>
      <button type="button" :class="btnCls" :disabled="busy || paused || uiStatus === 'syncing'" @click="emit('sync')">
        {{ $t('settings.mirrorSync') }}
      </button>
      <button
        type="button"
        class="btn-warn"
        :disabled="busy"
        @click="emit('remove')"
      >
        {{ $t('mirrorForm.remove') }}
      </button>
    </template>
  </div>
  </template>
</template>
