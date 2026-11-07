<script setup lang="ts">
withDefaults(
  defineProps<{
    glass?: boolean
    minWidthClass?: string
  }>(),
  {
    glass: true,
  },
)

const cellClasses =
  '[&_th]:bg-white/35 [&_th]:px-2 [&_th]:py-1.5 [&_th]:text-left [&_th]:align-baseline [&_th]:font-medium [&_td]:border-0 [&_td]:px-2 [&_td]:py-1.5 [&_td]:align-baseline [&_tbody_tr:nth-child(odd)]:bg-transparent [&_tbody_tr:nth-child(even)]:bg-row-alt [&_tbody_tr:hover]:bg-row-hover'
</script>

<template>
  <table
    :class="[
      'w-full border-collapse',
      glass ? 'glass overflow-hidden [&_th]:text-ink-muted' : '',
      minWidthClass,
      cellClasses,
    ]"
  >
    <slot />
  </table>
</template>
