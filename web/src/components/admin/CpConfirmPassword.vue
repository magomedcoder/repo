<script setup lang="ts">
import { ref } from 'vue'
import ModalForm from '@/components/ui/ModalForm.vue'

const props = defineProps<{
  title: string
  message?: string
}>()

const emit = defineEmits<{
  close: []
  confirm: [password: string]
}>()

const password = ref('')

function submit() {
  if (!password.value.trim()) return
  emit('confirm', password.value)
}
</script>

<template>
  <ModalForm
    :title="title"
    :submit-label="$t('common.ok')"
    @close="emit('close')"
    @submit="submit"
  >
    <p v-if="message" class="mb-3 text-sm text-ink-muted">{{ message }}</p>
    <label class="block text-sm">
      <span class="mb-1 block font-semibold">{{ $t('users.currentPassword') }}</span>
      <input
        v-model="password"
        type="password"
        class="field"
        required
        autocomplete="current-password"
      >
    </label>
  </ModalForm>
</template>
