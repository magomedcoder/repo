<script setup lang="ts">
import { ref } from 'vue'
import { formatDate } from '@/i18n'

const props = defineProps<{
  author: string
  body: string
  createdAt: string
  canEdit: boolean
  busy: boolean
}>()

const emit = defineEmits<{
  save: [body: string]
  remove: []
}>()

const editing = ref(false)
const draft = ref(props.body)

function startEdit() {
  draft.value = props.body
  editing.value = true
}

function save() {
  emit('save', draft.value)
  editing.value = false
}
</script>

<template>
  <article class="overflow-hidden rounded-md border border-line bg-white">
    <header class="flex items-center justify-between gap-2 border-b border-line bg-paper px-4 py-2 text-xs">
      <p>
        <span class="font-semibold text-ink">{{ author }}</span>
        <span class="text-ink-muted"> {{ formatDate(createdAt) }}</span>
      </p>
      <div v-if="canEdit" class="flex gap-3">
        <button
          type="button"
          class="text-ink-muted hover:text-accent"
          @click="startEdit"
        >{{ $t('issues.editComment') }}</button>
        <button
          type="button"
          class="text-warn"
          :disabled="busy" @click="emit('remove')"
        >{{ $t('issues.delete') }}</button>
      </div>
    </header>
    <form
      v-if="editing"
      class="space-y-2 p-4"
      @submit.prevent="save"
    >
      <textarea
        v-model="draft"
        class="input min-h-20 font-sans"
        required
      />
      <button
        type="submit"
        class="btn-ghost"
        :disabled="busy"
      >{{ $t('issues.saveComment') }}</button>
    </form>
    <p v-else class="whitespace-pre-wrap px-4 py-3 text-sm">{{ body }}</p>
  </article>
</template>
