<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { foldersApi, reposApi } from '@/api'
import type { Folder, MirrorDirection } from '@/api/types'
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
const mirrorEnabled = ref(false)
const mirrorDirection = ref<MirrorDirection>('pull')
const mirrorURL = ref('')
const mirrorUsername = ref('')
const mirrorPassword = ref('')
const error = ref('')
const busy = ref(false)

watch(mirrorEnabled, (on) => {
  if (!on) {
    mirrorDirection.value = 'pull'
  }
})

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
    const body: Parameters<typeof reposApi.create>[0] = {
      name: name.value.trim(),
      description: description.value.trim(),
      folder_path: folderKeyValue.value || undefined,
      private: isPrivate.value,
      show_on_home: !isPrivate.value && showOnHome.value,
      default_branch: defaultBranch.value.trim() || 'main',
    }
    if (mirrorEnabled.value && mirrorURL.value.trim()) {
      body.mirror_direction = mirrorDirection.value
      body.mirror_url = mirrorURL.value.trim()
      if (mirrorUsername.value.trim()) {
        body.mirror_username = mirrorUsername.value.trim()
      }

      if (mirrorPassword.value) {
        body.mirror_password = mirrorPassword.value
      }
    }
    
    await reposApi.create(body)
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
    :submit-label="busy ? (mirrorEnabled ? $t('modal.mirroring') : $t('modal.creating')) : (mirrorEnabled ? $t('modal.mirrorRepository') : $t('modal.createRepository'))"
    @close="emit('close')"
    @submit="submit"
  >
    <label class="block text-sm">
      <span class="mb-1 block font-medium">{{ $t('modal.name') }}</span>
      <input
        v-model="name"
        class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none"
        required
        maxlength="100"
        :placeholder="$t('modal.repoNamePlaceholder')"
        autofocus
      >
    </label>
    <label class="block text-sm">
      <span class="mb-1 block font-medium">{{ $t('modal.folder') }}</span>
      <select v-model="folderKeyValue" class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none">
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
        :class="['w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none', 'min-h-20 resize-y']"
        :placeholder="$t('modal.optional')"
      />
    </label>
    <label class="flex items-center gap-2 text-sm">
      <input v-model="mirrorEnabled" type="checkbox" class="accent-ink">
      {{ $t('modal.mirrorEnable') }}
    </label>
    <template v-if="mirrorEnabled">
      <label class="block text-sm">
        <span class="mb-1 block font-medium">{{ $t('modal.mirrorDirection') }}</span>
        <select v-model="mirrorDirection" class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none">
          <option value="pull">{{ $t('modal.mirrorPull') }}</option>
          <option value="push">{{ $t('modal.mirrorPush') }}</option>
        </select>
      </label>
      <p class="text-[85%] text-ink-muted">
        {{ mirrorDirection === 'push' ? $t('modal.mirrorPushHint') : $t('modal.mirrorPullHint') }}
      </p>
      <label class="block text-sm">
        <span class="mb-1 block font-medium">{{ $t('modal.mirrorURL') }}</span>
        <input
          v-model="mirrorURL"
          type="url"
          required
          class="w-full border border-line bg-paper px-2 py-1 font-mono text-[10pt] text-ink outline-none"
          :placeholder="$t('modal.mirrorURLPlaceholder')"
        >
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-medium">{{ $t('modal.mirrorUsername') }}</span>
        <input
          v-model="mirrorUsername"
          class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none"
          :placeholder="$t('modal.optional')"
          autocomplete="off"
        >
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-medium">{{ $t('modal.mirrorPassword') }}</span>
        <input
          v-model="mirrorPassword"
          type="password"
          class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none"
          :placeholder="$t('modal.optional')"
          autocomplete="new-password"
        >
      </label>
    </template>
    <label v-if="!mirrorEnabled || mirrorDirection === 'push'" class="block text-sm">
      <span class="mb-1 block font-medium">{{ $t('modal.defaultBranch') }}</span>
      <input
        v-model="defaultBranch"
        :class="['w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none', 'font-mono']"
        :placeholder="$t('modal.defaultBranchPlaceholder')"
      >
    </label>
    <label class="flex items-center gap-2 text-sm">
      <input
        v-model="isPrivate"
        type="checkbox"
        class="accent-ink"
        @change="isPrivate && (showOnHome = false)"
      >
      {{ $t('modal.private') }}
    </label>
    <label class="flex items-center gap-2 text-sm" :class="isPrivate && 'opacity-50'">
      <input
        v-model="showOnHome"
        type="checkbox"
        class="accent-ink"
        :disabled="isPrivate"
      >
      {{ $t('modal.showOnHome') }}
    </label>
    <p v-if="error" class="text-sm text-warn">{{ error }}</p>
  </ModalForm>
</template>
