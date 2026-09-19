<script setup lang="ts">
import { ref } from 'vue'
import type { Label } from '@/api/types'
import LabelPill from '@/components/ui/LabelPill.vue'

defineProps<{
  labels: Label[]
}>()

const emit = defineEmits<{
  create: [payload: {
    name: string
    color: string
  }]
  remove: [id: number]
}>()

const name = ref('')
const color = ref('#d73a4a')

function submit() {
  emit('create', {
    name: name.value,
    color: color.value
  })
  name.value = ''
}
</script>

<template>
  <section class="rounded-md border border-line bg-white p-4">
    <h2 class="mb-3 text-sm font-semibold">{{ $t('issues.labels') }}</h2>
    <ul class="mb-3 space-y-2">
      <li
        v-for="label in labels"
        :key="label.id"
        class="flex items-center justify-between gap-2"
      >
        <LabelPill :label="label" />
        <button
          type="button"
          class="text-xs text-warn"
          @click="emit('remove', label.id)"
        >{{ $t('issues.delete') }}</button>
      </li>
    </ul>
    <form class="flex flex-wrap gap-2" @submit.prevent="submit">
      <input
        v-model="name"
        class="input max-w-48"
        required
        :placeholder="$t('issues.labelName')"
      />
      <input
        v-model="color"
        type="color"
        class="h-8 w-10 rounded-md border border-line bg-white"
      />
      <button
        type="submit"
        class="btn-ghost"
      >{{ $t('issues.addLabel') }}</button>
    </form>
  </section>
</template>
