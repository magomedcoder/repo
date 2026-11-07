<script setup lang="ts">
import type { AccessLog } from '@/api/types'
import EmptyState from '@/components/ui/EmptyState.vue'
import Surface from '@/components/ui/Surface.vue'
import { formatDate } from '@/i18n'

defineProps<{
  items: AccessLog[]
}>()

function actionClass(action: string) {
  if (action.startsWith('admin_') || action === 'failed_auth') {
    return 'border-warn/40 bg-warn/10 text-warn'
  }

  if (action === 'login' || action === 'register' || action === 'token_create') {
    return 'border-ink/25 bg-white/60 text-ink'
  }

  if (action === 'push' || action === 'repo_create' || action === 'repo_delete' || action === 'repo_move') {
    return 'border-line-strong/80 bg-white/50 text-ink'
  }

  return 'border-line-strong/60 text-ink-muted'
}

function detailLabel(row: AccessLog) {
  const parts: string[] = []
  if (row.repo) {
    parts.push(row.repo)
  }

  if (row.ref) {
    parts.push(`@ ${row.ref}`)
  }

  if (row.path) {
    parts.push(row.path)
  }

  if (row.action.startsWith('admin_user_') && row.path) {
    return row.path
  }
  
  return parts.join(' ') || '-'
}
</script>

<template>
  <Surface class="overflow-x-auto">
    <table class="w-full min-w-[760px] border-collapse text-left text-[10pt]">
      <thead class="sticky top-0 bg-white/55 text-ink backdrop-blur-sm">
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
          class="align-top odd:bg-transparent even:bg-row-alt"
        >
          <td class="border-b border-line px-2 py-1.5 whitespace-nowrap text-ink-muted">
            {{ formatDate(row.created_at) }}
          </td>
          <td class="border-b border-line px-2 py-1.5">
            <span
              class="inline-flex rounded-full border px-1.5 py-px font-mono text-[90%] font-medium"
              :class="actionClass(row.action)"
            >
              {{ row.action }}
            </span>
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
            {{ detailLabel(row) }}
          </td>
        </tr>
        <tr v-if="!items.length">
          <EmptyState
            as="td"
            :colspan="6"
            :message="$t('accessLogs.empty')"
          />
        </tr>
      </tbody>
    </table>
  </Surface>
</template>
