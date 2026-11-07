<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    variant?: 'default' | 'primary' | 'warn' | 'ghost'
    type?: 'button' | 'submit' | 'reset'
    disabled?: boolean
  }>(),
  {
    variant: 'default',
    type: 'button',
    disabled: false,
  },
)

const variantClass = computed(() => {
  switch (props.variant) {
    case 'primary':
      return 'btn-primary'
    case 'warn':
      return 'btn-warn'
    case 'ghost':
      return 'btn-ghost'
    default:
      return 'btn'
  }
})
</script>

<template>
  <button
    :type="type"
    :disabled="disabled"
    :class="variantClass"
  >
    <slot />
  </button>
</template>
