<script setup lang="ts">
import { field } from '@/lib/classes'
import { onMounted, ref } from 'vue'
import { foldersApi, reposApi } from '@/api'
import type { Folder } from '@/api/types'
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
const folderKey = ref(props.folderId == null ? '' : String(props.folderId))
const folders = ref<Folder[]>([])
const description = ref('')
const isPrivate = ref(true)
const showOnHome = ref(false)
const defaultBranch = ref('main')
const error = ref('')
const busy = ref(false)

onMounted(async () => {
  try {
    const res = await foldersApi.list()
    folders.value = res.folders ?? []
  } catch {
    folders.value = []
  }
})

async function submit() {
  error.value = ''
  busy.value = true
  try {
    await reposApi.create({
      name: name.value.trim(),
      description: description.value.trim(),
      folder_id: folderKey.value === '' ? null : Number(folderKey.value),
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
        :class="field"
        required
        maxlength="100"
        :placeholder="$t('modal.repoNamePlaceholder')"
        autofocus
      />
    </label>
    <label class="block text-sm">
      <span class="mb-1 block font-medium">{{ $t('modal.folder') }}</span>
      <select v-model="folderKey" :class="field">
        <option value="">{{ $t('nav.root') }}</option>
        <option
          v-for="folder in folders"
          :key="folder.id"
          :value="String(folder.id)"
        >
          {{ folder.name }}
        </option>
      </select>
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
        :placeholder="$t('modal.defaultBranchPlaceholder')"
      />
    </label>
    <label class="flex items-center gap-2 text-sm">
      <input
        v-model="isPrivate"
        type="checkbox"
        class="accent-ink"
        @change="isPrivate && (showOnHome = false)"
      />
      {{ $t('modal.private') }}
    </label>
    <label class="flex items-center gap-2 text-sm" :class="isPrivate && 'opacity-50'">
      <input
        v-model="showOnHome"
        type="checkbox"
        class="accent-ink"
        :disabled="isPrivate"
      />
      {{ $t('modal.showOnHome') }}
    </label>
    <p v-if="error" class="text-sm text-warn">{{ error }}</p>
  </ModalForm>
</template>
