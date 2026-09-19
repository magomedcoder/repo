<script setup lang="ts">
import type { Label } from '@/api/types'
import LabelPill from '@/components/ui/LabelPill.vue'

defineProps<{
  labels: Label[]
}>()

const selected = defineModel<number[]>({ required: true })
</script>

<template>
  <div
    v-if="labels.length"
    class="flex flex-wrap gap-x-3 gap-y-2"
  >
    <label
      v-for="label in labels"
      :key="label.id"
      class="inline-flex items-center gap-1.5 text-sm"
    >
      <input
        v-model="selected"
        type="checkbox"
        :value="label.id"
        class="accent-[#1f883d]"
      />
      <LabelPill :label="label" />
    </label>
  </div>
</template>
