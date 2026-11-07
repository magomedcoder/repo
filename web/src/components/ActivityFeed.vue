<script setup lang="ts">
import type { ActivityEvent } from '@/api/types'
import { formatDate } from '@/i18n'
import Icon from '@/components/ui/Icon.vue'
import GlassTable from '@/components/ui/GlassTable.vue'

defineProps<{
  events: ActivityEvent[]
  loading?: boolean
  error?: string
}>()
</script>

<template>
  <div>
    <h2 class="mb-2 inline-flex items-center gap-1.5 font-bold text-ink"><Icon name="clock" size="sm" class="text-ink-muted" />{{ $t('activity.title') }}</h2>
    <p v-if="loading" class="text-ink-muted">{{ $t('common.loading') }}</p>
    <p v-else-if="error" class="text-warn">{{ error }}</p>
    <GlassTable v-else-if="events.length">
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
    </GlassTable>
    <p v-else class="text-ink-muted">{{ $t('activity.empty') }}</p>
  </div>
</template>
