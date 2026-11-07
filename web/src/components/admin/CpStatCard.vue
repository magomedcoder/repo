<script setup lang="ts">
import { RouterLink } from 'vue-router'
import Icon from '@/components/ui/Icon.vue'
import type { IconName } from '@/components/ui/Icon.vue'

defineProps<{
  label: string
  value: string | number
  icon: IconName
  to?: { 
    name: string
    query?: Record<string, string> 
  }
  tone?: 'default' | 'warn'
}>()
</script>

<template>
  <component
    :is="to ? RouterLink : 'div'"
    :to="to"
    class="glass flex items-center gap-3 px-3 py-3 no-underline transition-colors"
    :class="to ? 'hover:bg-row-hover' : ''"
  >
    <span
      class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full border"
      :class="tone === 'warn' ? 'border-warn/40 bg-warn/10 text-warn' : 'border-line-strong/70 bg-white/40 text-ink-muted'"
    >
      <Icon :name="icon" size="md" />
    </span>
    <div class="min-w-0">
      <p class="text-[11px] font-medium uppercase tracking-wide text-ink-muted">{{ label }}</p>
      <p class="mt-0.5 text-[1.25rem] font-semibold leading-none text-ink">{{ value }}</p>
    </div>
  </component>
</template>
