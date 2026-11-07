<script setup lang="ts">
defineProps<{
  cols?: 2 | 3
}>()
</script>

<template>
  <div
    class="page-card mb-3"
    :class="{
      'grid gap-2 sm:grid-cols-2': cols === 2,
      'grid gap-2 sm:grid-cols-3': cols === 3,
    }"
  >
    <slot />
  </div>
</template>
