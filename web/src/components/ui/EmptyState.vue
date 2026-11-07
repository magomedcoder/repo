<script setup lang="ts">
withDefaults(
  defineProps<{
    as?: 'div' | 'p' | 'li' | 'td' | 'span'
    colspan?: number
    message?: string
  }>(),
  {
    as: 'div',
  },
)
</script>

<template>
  <component
    :is="as"
    :colspan="as === 'td' ? colspan : undefined"
    class="px-4 py-10 text-center text-sm text-ink-muted"
  >
    <slot>{{ message }}</slot>
  </component>
</template>
