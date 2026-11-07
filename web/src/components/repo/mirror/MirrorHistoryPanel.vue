<script setup lang="ts">
import type { MirrorDirection, MirrorDryRun, MirrorSyncEvent } from '@/api/types'
import { formatDate } from '@/i18n'

defineProps<{
  configured: boolean
  direction: MirrorDirection
  events: MirrorSyncEvent[]
  historyLoading: boolean
  historyError: string
  dryRun: MirrorDryRun | null
  dryRunBusy: boolean
  dryRunError: string
  dryRunHasAction: boolean
  btnCls: string
  triggerLabel: (trigger: string) => string
  eventStatusLabel: (status: string) => string
  formatDuration: (ms: number) => string
  actionLabel: (action: string) => string
  shortSha: (sha?: string) => string
}>()

const emit = defineEmits<{
  refreshHistory: []
  dryRun: []
}>()
</script>

<template>
  <template v-if="configured">
    <div class="surface-header justify-between border-y border-x-0">
      <span>{{ $t('mirrorForm.historyTitle') }}</span>
      <button type="button" :class="btnCls" :disabled="historyLoading" @click="emit('refreshHistory')">
        {{ historyLoading ? $t('common.loading') : $t('mirrorForm.refresh') }}
      </button>
    </div>
    <p v-if="historyError" class="px-3 pt-3 text-sm text-warn sm:px-4">{{ historyError }}</p>
    <div class="overflow-x-auto">
      <table class="w-full min-w-[640px] border-collapse text-left text-[10pt]">
        <thead class="bg-white/35 text-ink">
          <tr>
            <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('mirrorForm.colStarted') }}</th>
            <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('mirrorForm.colFinished') }}</th>
            <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('mirrorForm.colTrigger') }}</th>
            <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('mirrorForm.colResult') }}</th>
            <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('mirrorForm.colDetails') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="ev in events" :key="ev.id" class="align-top odd:bg-transparent even:bg-row-alt">
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
      <div class="surface-header justify-between border-y border-x-0">
        <span>{{ $t('mirrorForm.dryRunTitle') }}</span>
        <button type="button" :class="btnCls" :disabled="dryRunBusy" @click="emit('dryRun')">
          {{ dryRunBusy ? $t('mirrorForm.dryRunning') : $t('mirrorForm.dryRunRun') }}
        </button>
      </div>
      <div class="p-3 sm:p-4">
        <p v-if="dryRunError" class="text-sm text-warn">{{ dryRunError }}</p>
        <p v-else-if="!dryRun && !dryRunBusy" class="text-xs text-ink-muted">{{ $t('mirrorForm.dryRunHint') }}</p>
        <template v-else-if="dryRun">
          <p v-if="dryRun.head || dryRun.default_branch" class="mb-2 text-xs text-ink-muted">{{ $t('mirror.remoteHead', { ref: dryRun.head || dryRun.default_branch }) }}</p>
          <p v-if="!dryRun.refs?.length" class="text-sm text-ink-muted">{{ $t('mirrorForm.dryRunEmpty') }}</p>
          <div v-else class="surface overflow-x-auto">
            <table class="w-full min-w-[520px] border-collapse text-left text-[10pt]">
              <thead class="bg-white/35 text-ink">
                <tr>
                  <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('mirrorForm.colRef') }}</th>
                  <th v-if="dryRunHasAction" class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('mirrorForm.colAction') }}</th>
                  <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('mirrorForm.colNew') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="r in dryRun.refs" :key="r.name" class="odd:bg-transparent even:bg-row-alt">
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
</template>
