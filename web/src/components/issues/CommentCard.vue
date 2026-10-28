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
  <article class="overflow-hidden border border-line bg-paper">
    <header class="flex items-center gap-2 border-b border-line bg-path px-3 py-1.5 text-[10pt] text-ink justify-between">
      <p>
        <span class="font-semibold text-ink">{{ author }}</span>
        <span class="font-normal text-ink-muted"> {{ formatDate(createdAt) }}</span>
      </p>
      <div v-if="canEdit" class="flex gap-3 font-normal normal-case tracking-normal">
        <button
          type="button"
          class="text-ink-muted hover:text-ink"
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
      class="p-3 sm:p-4 space-y-2"
      @submit.prevent="save"
    >
      <textarea
        v-model="draft"
        :class="['w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none', 'min-h-20']"
        required
      />
      <button
        type="submit"
        class="inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-ink hover:bg-row-hover disabled:cursor-not-allowed disabled:opacity-50"
        :disabled="busy"
      >{{ $t('issues.saveComment') }}</button>
    </form>
    <p v-else class="whitespace-pre-wrap px-4 py-3 text-sm">{{ body }}</p>
  </article>
</template>
