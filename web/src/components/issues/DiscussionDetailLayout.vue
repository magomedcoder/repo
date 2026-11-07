<script setup lang="ts">
defineProps<{
  withSidebar?: boolean
}>()
</script>

<template>
  <div>
    <div class="mb-4 flex flex-wrap items-start justify-between gap-3">
      <div>
        <h1 class="text-[140%] font-bold text-ink">
          <slot name="title" />
        </h1>
        <slot name="subtitle" />
      </div>
      <div
        v-if="$slots.actions"
        class="flex flex-wrap gap-2"
      >
        <slot name="actions" />
      </div>
    </div>

    <div
      v-if="$slots.meta"
      class="mb-4 flex flex-wrap items-center gap-2 text-sm"
    >
      <slot name="meta" />
    </div>

    <div
      :class="withSidebar || $slots.sidebar
        ? 'grid gap-4 lg:grid-cols-[minmax(0,1fr)_240px]'
        : ''"
    >
      <div class="space-y-4">
        <slot name="body" />
        <slot name="before-comments" />
        <slot name="comments" />
        <slot name="comment-form" />
        <slot />
      </div>
      <aside
        v-if="$slots.sidebar"
        class="space-y-4"
      >
        <slot name="sidebar" />
      </aside>
    </div>
  </div>
</template>
