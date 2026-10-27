<script setup lang="ts">
import type { ActivityEvent } from '@/api/types'
import { formatDate } from '@/i18n'
import { dataTable } from '@/lib/classes'

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
    <table v-else-if="events.length" :class="dataTable">
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
            {{ ev.summary }}
            <span v-if="ev.ref" class="font-mono text-ink-muted"> {{ ev.ref }}</span>
          </td>
        </tr>
      </tbody>
    </table>
    <p v-else class="text-ink-muted">{{ $t('activity.empty') }}</p>
  </div>
</template>
