<script setup lang="ts">
import { ref } from 'vue'
import { foldersApi } from '@/api'
import ModalForm from '@/components/ModalForm.vue'
import { localizeError } from '@/i18n'

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
    error.value = localizeError(err, 'errors.createFolder')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <ModalForm :title="$t('modal.newFolder')" :submit-label="busy ? $t('modal.creating') : $t('modal.createFolder')" @close="emit('close')" @submit="submit">
    <label class="block text-sm">
      <span class="mb-1 block font-medium">{{ $t('modal.name') }}</span>
      <input v-model="name" class="input" required maxlength="64" placeholder="work" autofocus />
    </label>
    <p v-if="error" class="text-sm text-warn">{{ error }}</p>
  </ModalForm>
</template>
