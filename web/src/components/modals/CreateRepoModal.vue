<script setup lang="ts">
import { ref } from 'vue'
import { reposApi } from '@/api'
import ModalForm from '@/components/ui/ModalForm.vue'
import { localizeError } from '@/i18n'

const emit = defineEmits<{
  close: []
  created: []
}>()

const name = ref('')
const description = ref('')
const isPrivate = ref(true)
const showOnHome = ref(false)
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
      private: isPrivate.value,
      show_on_home: !isPrivate.value && showOnHome.value,
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
    :busy="busy"
    :submit-label="busy ? $t('modal.creating') : $t('modal.createRepository')"
    @close="emit('close')"
    @submit="submit"
  >
    <label class="block text-sm">
      <span class="mb-1 block font-medium">{{ $t('modal.name') }}</span>
      <input
        v-model="name"
        :disabled="busy"
        class="field"
        required
        maxlength="100"
        :placeholder="$t('modal.repoNamePlaceholder')"
        autofocus
      >
    </label>
    <label class="block text-sm">
      <span class="mb-1 block font-medium">{{ $t('modal.description') }}</span>
      <textarea
        v-model="description"
        rows="3"
        :disabled="busy"
        class="field min-h-20 resize-y"
        :placeholder="$t('modal.optional')"
      />
    </label>
    <label class="block text-sm">
      <span class="mb-1 block font-medium">{{ $t('modal.defaultBranch') }}</span>
      <input
        v-model="defaultBranch"
        :disabled="busy"
        class="field font-mono"
        :placeholder="$t('modal.defaultBranchPlaceholder')"
      >
    </label>
    <label class="flex items-center gap-2 text-sm">
      <input
        v-model="isPrivate"
        type="checkbox"
        class="accent-ink"
        :disabled="busy"
        @change="isPrivate && (showOnHome = false)"
      >
      {{ $t('modal.private') }}
    </label>
    <label class="flex items-center gap-2 text-sm" :class="isPrivate && 'opacity-50'">
      <input
        v-model="showOnHome"
        type="checkbox"
        class="accent-ink"
        :disabled="busy || isPrivate"
      >
      {{ $t('modal.showOnHome') }}
    </label>
    <p v-if="error" class="text-sm text-warn">{{ error }}</p>
  </ModalForm>
</template>
