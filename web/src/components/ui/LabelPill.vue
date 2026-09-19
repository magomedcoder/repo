<script setup lang="ts">
import type { Label } from '@/api/types'

defineProps<{
  label: Label
}>()

function textOn(color: string) {
  const hex = color.replace('#', '')
  if (hex.length !== 6) {
    return '#fff'
  }

  const r = Number.parseInt(hex.slice(0, 2), 16)
  const g = Number.parseInt(hex.slice(2, 4), 16)
  const b = Number.parseInt(hex.slice(4, 6), 16)

  return (0.299 * r + 0.587 * g + 0.114 * b) / 255 > 0.62 ? '#1f2328' : '#fff'
}
</script>

<template>
  <span
    class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium leading-4"
    :style="{
      background: label.color,
      color: textOn(label.color)
    }"
  >
    {{ label.name }}
  </span>
</template>
