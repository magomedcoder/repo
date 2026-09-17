<script setup lang="ts">
import { ref } from 'vue'
import { foldersApi } from '@/api'
import ModalForm from '@/components/ModalForm.vue'
import { ApiError } from '@/api/client'

const props = defineProps<{
  parentId?: number | null
}>()

const emit = defineEmits<{
  close: []
  created: []
}>()

const name = ref('')
const error = ref('')
const busy = ref(false)

async function submit() {
  error.value = ''
  busy.value = true
  try {
    await foldersApi.create({
      name: name.value.trim(),
      parent_id: props.parentId ?? null,
    })
    emit('created')
    emit('close')
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'Failed to create folder'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <ModalForm title="New folder" :submit-label="busy ? 'Creating...' : 'Create folder'" @close="emit('close')" @submit="submit">
    <label class="block text-sm">
      <span class="mb-1 block font-medium">Name</span>
      <input v-model="name" class="input" required maxlength="64" placeholder="work" autofocus />
    </label>
    <p v-if="error" class="text-sm text-warn">{{ error }}</p>
  </ModalForm>
</template>
