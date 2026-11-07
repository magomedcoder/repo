<script setup lang="ts">
import Icon from '@/components/ui/Icon.vue'
import type { IconName } from '@/components/ui/Icon.vue'

defineProps<{
  title: string
  icon?: IconName
  hint?: string
}>()
</script>

<template>
  <div class="page-card mb-4 flex flex-wrap items-start justify-between gap-3">
    <div>
      <h1 class="inline-flex items-center gap-2 text-[140%] font-bold text-ink">
        <Icon
          v-if="icon"
          :name="icon"
          class="text-ink-muted"
        />
        {{ title }}
      </h1>
      <p
        v-if="hint || $slots.hint"
        class="mt-1 text-sm text-ink-muted"
      >
        <slot name="hint">{{ hint }}</slot>
      </p>
    </div>
    <div
      v-if="$slots.actions"
      class="flex flex-wrap items-center gap-2"
    >
      <slot name="actions" />
    </div>
  </div>
</template>
