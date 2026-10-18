<script setup lang="ts">
import { field } from '@/lib/classes'
import { ref } from 'vue'
import { reposApi } from '@/api'
import ModalForm from '@/components/ModalForm.vue'
import { localizeError } from '@/i18n'

const props = defineProps<{
  folderId?: number | null
}>()

const emit = defineEmits<{
  close: []
  created: []
}>()

const name = ref('')
const path = ref('')
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
      path: path.value.trim(),
      description: description.value.trim(),
      folder_id: props.folderId ?? null,
      private: isPrivate.value,
      default_branch: defaultBranch.value.trim() || 'main',
    })
    emit('created')
    emit('close')
  } catch (err) {
    error.value = localizeError(err, 'errors.createRepository')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <ModalForm
    :title="$t('modal.newRepository')"
    :submit-label="busy ? $t('modal.creating') : $t('modal.createRepository')"
    @close="emit('close')"
    @submit="submit"
  >
    <label class="block text-sm">
      <span class="mb-1 block font-medium">{{ $t('modal.name') }}</span>
      <input
        v-model="name"
        :class="field"
        :required="!path"
        maxlength="100"
        placeholder="my-repo"
        autofocus
      />
    </label>
    <label class="block text-sm">
      <span class="mb-1 block font-medium">{{ $t('modal.path') }}</span>
      <input
        v-model="path"
        :class="[field, 'font-mono']"
        placeholder="notes/diary.git"
      />
    </label>
    <label class="block text-sm">
      <span class="mb-1 block font-medium">{{ $t('modal.description') }}</span>
      <input
        v-model="description"
        :class="field"
        :placeholder="$t('modal.optional')"
      />
    </label>
    <label class="block text-sm">
      <span class="mb-1 block font-medium">{{ $t('modal.defaultBranch') }}</span>
      <input
        v-model="defaultBranch"
        :class="[field, 'font-mono']"
      />
    </label>
    <label class="flex items-center gap-2 text-sm">
      <input
        v-model="isPrivate"
        type="checkbox"
        class="accent-moss"
      />
      {{ $t('modal.private') }}
    </label>
    <p v-if="error" class="text-sm text-warn">{{ error }}</p>
  </ModalForm>
</template>
