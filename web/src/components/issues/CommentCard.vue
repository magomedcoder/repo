<script setup lang="ts">
import { btnGhost, field } from '@/lib/classes'
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
  <article class="term-block">
    <header class="term-block-hd justify-between">
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
      class="term-block-bd space-y-2"
      @submit.prevent="save"
    >
      <textarea
        v-model="draft"
        :class="[field, 'min-h-20']"
        required
      />
      <button
        type="submit"
        :class="btnGhost"
        :disabled="busy"
      >{{ $t('issues.saveComment') }}</button>
    </form>
    <p v-else class="whitespace-pre-wrap px-4 py-3 text-sm">{{ body }}</p>
  </article>
</template>
