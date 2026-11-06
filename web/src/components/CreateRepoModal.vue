<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { foldersApi, reposApi } from '@/api'
import type { Folder } from '@/api/types'
import ModalForm from '@/components/ModalForm.vue'
import { folderKey } from '@/lib/folderNav'
import { normalizeRepoPath } from '@/lib/repoPath'
import { localizeError } from '@/i18n'

const props = defineProps<{
  folderPath?: string | null
}>()

const emit = defineEmits<{
  close: []
  created: []
}>()

const name = ref('')
const folderKeyValue = ref('')
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
    const want = props.folderPath ? normalizeRepoPath(props.folderPath) : ''
    if (want) {
      const match = folders.value.find((f) => folderKey(f) === want)
      if (match) {
        folderKeyValue.value = folderKey(match)
      }
    }
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
      folder_path: folderKeyValue.value || undefined,
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
        class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none"
        required
        maxlength="100"
        :placeholder="$t('modal.repoNamePlaceholder')"
        autofocus
      >
    </label>
    <label class="block text-sm">
      <span class="mb-1 block font-medium">{{ $t('modal.folder') }}</span>
      <select v-model="folderKeyValue" :disabled="busy" class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none">
        <option value="">{{ $t('nav.root') }}</option>
        <option
          v-for="folder in folders"
          :key="folder.path"
          :value="folderKey(folder)"
        >
          {{ folder.name }}
        </option>
      </select>
    </label>
    <label class="block text-sm">
      <span class="mb-1 block font-medium">{{ $t('modal.description') }}</span>
      <textarea
        v-model="description"
        rows="3"
        :disabled="busy"
        class="min-h-20 w-full resize-y border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none"
        :placeholder="$t('modal.optional')"
      />
    </label>
    <label class="block text-sm">
      <span class="mb-1 block font-medium">{{ $t('modal.defaultBranch') }}</span>
      <input
        v-model="defaultBranch"
        :disabled="busy"
        class="w-full border border-line bg-paper px-2 py-1 font-mono text-[10pt] text-ink outline-none"
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
