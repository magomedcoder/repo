<script setup lang="ts">
defineEmits<{
  submit: []
}>()
</script>

<template>
  <form
    class="page-card page-enter w-full max-w-sm space-y-4 self-center"
    @submit.prevent="$emit('submit')"
  >
    <slot />
  </form>
</template>
