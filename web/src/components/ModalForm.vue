<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  title: string
  submitLabel?: string
}>()

const emit = defineEmits<{
  close: []
  submit: []
}>()

const open = ref(true)

function onBackdrop() {
  emit('close')
}
</script>

<template>
  <div
    v-if="open"
    class="fixed inset-0 z-50 flex items-center justify-center bg-ink/40 p-4 backdrop-blur-[2px]"
    @click.self="onBackdrop"
  >
    <div class="panel w-full max-w-md p-5 shadow-lg" role="dialog" aria-modal="true">
      <div class="mb-4 flex items-start justify-between gap-3">
        <h2 class="font-display text-xl font-bold">{{ title }}</h2>
        <button type="button" class="text-ink-muted hover:text-ink" aria-label="Close" @click="emit('close')">
          ✕
        </button>
      </div>
      <form class="space-y-3" @submit.prevent="emit('submit')">
        <slot />
        <div class="flex justify-end gap-2 pt-2">
          <button type="button" class="btn-ghost" @click="emit('close')">Cancel</button>
          <button type="submit" class="btn-primary">{{ submitLabel || 'Create' }}</button>
        </div>
      </form>
    </div>
  </div>
</template>
