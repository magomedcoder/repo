<script setup lang="ts">
defineProps<{
  title: string
  description?: string
}>()
</script>

<template>
  <section class="space-y-4">
    <header class="flex flex-wrap items-end justify-between gap-3">
      <div class="min-w-0">
        <h2 class="text-[1.05rem] font-semibold leading-snug text-ink">{{ title }}</h2>
        <p v-if="description" class="mt-0.5 text-[12px] text-ink-muted">{{ description }}</p>
      </div>
      <div v-if="$slots.actions" class="flex flex-wrap items-center gap-1.5">
        <slot name="actions" />
      </div>
    </header>
    <slot />
  </section>
</template>
