<script setup lang="ts">
import type { AccessLog } from '@/api/types'
import EmptyState from '@/components/ui/EmptyState.vue'
import Surface from '@/components/ui/Surface.vue'
import { formatDate } from '@/i18n'

defineProps<{
  items: AccessLog[]
}>()
</script>

<template>
  <Surface class="overflow-x-auto">
    <table class="w-full min-w-[760px] border-collapse text-left text-[10pt]">
      <thead class="bg-white/35 text-ink">
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
          <td class="border-b border-line px-2 py-1.5 font-mono text-[90%]">
            {{ row.action }}
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
            <span v-if="row.repo">{{ row.repo }}</span>
            <span v-if="row.ref"> @ {{ row.ref }}</span>
            <span v-if="row.path">{{ row.path }}</span>
            <span v-if="!row.repo && !row.ref && !row.path">-</span>
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
