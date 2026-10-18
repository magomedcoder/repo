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
  <section class="overflow-hidden rounded-md border border-line bg-white">
    <h2 class="border-b border-line bg-paper px-4 py-2 text-sm font-semibold">
      {{ $t('activity.title') }}
    </h2>
    <p v-if="loading" class="px-4 py-6 text-sm text-ink-muted">{{ $t('common.loading') }}</p>
    <p v-else-if="error" class="px-4 py-6 text-sm text-warn">{{ error }}</p>
    <ul v-else-if="events.length" class="divide-y divide-line">
      <li v-for="ev in events" :key="ev.id" class="px-4 py-3 text-sm">
        <p>
          <span class="font-semibold">{{ ev.actor }}</span>
          <span class="text-ink-muted"> {{ ev.owner }}</span>
        </p>
        <p class="mt-0.5">{{ ev.summary }}</p>
        <p class="mt-1 text-xs text-ink-muted">
          <span v-if="ev.ref" class="font-mono">{{ ev.ref }} </span>
          {{ formatDate(ev.created_at) }}
        </p>
      </li>
    </ul>
    <p v-else class="px-4 py-8 text-center text-sm text-ink-muted">{{ $t('activity.empty') }}</p>
  </section>
</template>
