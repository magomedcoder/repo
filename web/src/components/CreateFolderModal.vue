<script setup lang="ts">
import { ref } from 'vue'
import { foldersApi } from '@/api'
import ModalForm from '@/components/ModalForm.vue'
import { localizeError } from '@/i18n'

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
    })
    emit('created')
    emit('close')
  } catch (err) {
    error.value = localizeError(err, 'errors.createFolder')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <ModalForm :title="$t('modal.newFolder')" :busy="busy" :submit-label="busy ? $t('modal.creating') : $t('modal.createFolder')" @close="emit('close')" @submit="submit">
    <label class="block text-sm">
      <span class="mb-1 block font-medium">{{ $t('modal.name') }}</span>
      <input v-model="name" class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none" required maxlength="64" :placeholder="$t('modal.folderNamePlaceholder')" autofocus />
    </label>
    <p v-if="error" class="text-sm text-warn">{{ error }}</p>
  </ModalForm>
</template>
