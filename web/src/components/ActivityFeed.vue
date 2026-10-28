<script setup lang="ts">
import type { ActivityEvent } from '@/api/types'
import { formatDate } from '@/i18n'

defineProps<{
  events: ActivityEvent[]
  loading?: boolean
  error?: string
}>()
</script>

<template>
  <div>
    <h2 class="mb-2 font-bold text-ink">{{ $t('activity.title') }}</h2>
    <p v-if="loading" class="text-ink-muted">{{ $t('common.loading') }}</p>
    <p v-else-if="error" class="text-warn">{{ error }}</p>
    <table v-else-if="events.length" class="w-full border-collapse [&_th]:px-2 [&_th]:py-1 [&_th]:text-left [&_th]:align-baseline [&_th]:font-medium [&_th]:text-ink-muted [&_td]:border-0 [&_td]:px-2 [&_td]:py-1 [&_td]:align-baseline [&_tbody_tr:nth-child(odd)]:bg-paper [&_tbody_tr:nth-child(even)]:bg-row-alt [&_tbody_tr:hover]:bg-row-hover">
      <thead>
        <tr>
          <th>{{ $t('index.age') }}</th>
          <th>{{ $t('index.message') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="ev in events" :key="ev.id">
          <td class="whitespace-nowrap text-ink-muted">{{ formatDate(ev.created_at) }}</td>
          <td>
            <span class="font-bold">{{ ev.actor }}</span>
            {{ $te(`activity.types.${ev.type}`) ? $t(`activity.types.${ev.type}`) : ev.summary }}
            <span v-if="ev.ref" class="font-mono text-ink-muted"> {{ ev.ref }}</span>
          </td>
        </tr>
      </tbody>
    </table>
    <p v-else class="text-ink-muted">{{ $t('activity.empty') }}</p>
  </div>
</template>
