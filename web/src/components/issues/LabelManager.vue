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
  <section class="overflow-hidden border border-line bg-paper">
    <h2 class="flex items-center gap-2 border-b border-line bg-path px-3 py-1.5 text-[10pt] text-ink">{{ $t('issues.labels') }}</h2>
    <div class="p-3 sm:p-4">
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
          :class="['w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none', 'max-w-48']"
          required
          :placeholder="$t('issues.labelName')"
        />
        <input
          v-model="color"
          type="color"
          class="h-8 w-10 border border-line bg-paper"
        />
        <button
          type="submit"
          class="inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-ink hover:bg-row-hover disabled:cursor-not-allowed disabled:opacity-50"
        >{{ $t('issues.addLabel') }}</button>
      </form>
    </div>
  </section>
</template>
