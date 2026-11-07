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
  if (row.action.startsWith('admin_user_') && row.path) {
    return row.path
  }

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

  return parts.join(' ')
}
</script>

<template>
  <Surface class="overflow-hidden">
    <EmptyState
      v-if="!items.length"
      :message="$t('accessLogs.empty')"
    />
    <ul v-else class="divide-y divide-line-strong/30">
      <li
        v-for="row in items"
        :key="row.id"
        class="px-3 py-2.5 odd:bg-transparent even:bg-row-alt sm:px-4"
      >
        <div class="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1">
          <div class="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
            <span
              class="inline-flex shrink-0 rounded-full border px-1.5 py-px font-mono text-[10px] font-medium"
              :class="actionClass(row.action)"
            >
              {{ row.action }}
            </span>
            <span class="truncate font-semibold text-ink">
              {{ row.username || '' }}
            </span>
            <span
              v-if="detailLabel(row)"
              class="min-w-0 truncate text-[12px] text-ink-muted"
              :title="detailLabel(row)"
            >
              {{ detailLabel(row) }}
            </span>
          </div>
          <time class="shrink-0 text-[12px] text-ink-muted">
            {{ formatDate(row.created_at) }}
          </time>
        </div>
        <div class="mt-1 flex min-w-0 flex-wrap items-baseline gap-x-3 gap-y-0.5 text-[11px] text-ink-muted">
          <span
            v-if="row.ip"
            class="font-mono"
            :title="row.ip"
          >{{ row.ip }}</span>
          <span
            v-if="row.user_agent"
            class="min-w-0 truncate font-mono"
            :title="row.user_agent"
          >{{ row.user_agent }}</span>
        </div>
      </li>
    </ul>
  </Surface>
</template>
