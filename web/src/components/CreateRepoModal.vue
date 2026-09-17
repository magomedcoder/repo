<script setup lang="ts">
import { ref } from 'vue'
import { reposApi } from '@/api'
import ModalForm from '@/components/ModalForm.vue'
import { ApiError } from '@/api/client'

const props = defineProps<{
  folderId?: number | null
}>()

const emit = defineEmits<{
  close: []
  created: []
}>()

const name = ref('')
const description = ref('')
const isPrivate = ref(false)
const defaultBranch = ref('main')
const error = ref('')
const busy = ref(false)

async function submit() {
  error.value = ''
  busy.value = true
  try {
    await reposApi.create({
      name: name.value.trim(),
      description: description.value.trim(),
      folder_id: props.folderId ?? null,
      private: isPrivate.value,
      default_branch: defaultBranch.value.trim() || 'main',
    })
    emit('created')
    emit('close')
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'Failed to create repository'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <ModalForm
    title="New repository"
    :submit-label="busy ? 'Creating...' : 'Create repository'"
    @close="emit('close')"
    @submit="submit"
  >
    <label class="block text-sm">
      <span class="mb-1 block font-medium">Name</span>
      <input
        v-model="name"
        class="input"
        required
        maxlength="100"
        placeholder="my-repo"
        autofocus
      />
    </label>
    <label class="block text-sm">
      <span class="mb-1 block font-medium">Description</span>
      <input
        v-model="description"
        class="input"
        placeholder="Optional"
      />
    </label>
    <label class="block text-sm">
      <span class="mb-1 block font-medium">Default branch</span>
      <input
        v-model="defaultBranch"
        class="input font-mono"
      />
    </label>
    <label class="flex items-center gap-2 text-sm">
      <input
        v-model="isPrivate"
        type="checkbox"
        class="accent-moss"
      />
      Private repository
    </label>
    <p v-if="error" class="text-sm text-warn">{{ error }}</p>
  </ModalForm>
</template>
