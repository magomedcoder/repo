<script setup lang="ts">
import { btnDanger, btnPrimary, field } from '@/lib/classes'
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { foldersApi, reposApi } from '@/api'
import PageState from '@/components/ui/PageState.vue'
import { repoPathMatches, useInjectedRepo } from '@/composables/useInjectedRepo'
import type { PageError } from '@/i18n'
import { i18n, localizeError } from '@/i18n'

const props = defineProps<{
  repoPath: string
}>()

const router = useRouter()
const repo = useInjectedRepo()

const description = ref('')
const isPrivate = ref(false)
const protectBranch = ref(true)
const defaultBranch = ref('main')
const folderKey = ref('')
const folderOptions = ref<{ id: string; label: string }[]>([{
  id: '',
  label: i18n.global.t('nav.root'),
}])
const loadError = ref<PageError | string>('')
const formError = ref('')
const message = ref('')
const dataLoading = ref(false)
const saving = ref(false)
const deleting = ref(false)

const loading = computed(
  () => !repo.value || !repoPathMatches(repo.value, props.repoPath) || dataLoading.value,
)

function parseFolderKey(key: string): number | null {
  return key === '' ? null : Number(key)
}

function syncFormFromRepo() {
  if (!repo.value) {
    return
  }
  description.value = repo.value.description
  isPrivate.value = repo.value.is_private
  protectBranch.value = repo.value.protect_default_branch !== false
  defaultBranch.value = repo.value.default_branch
  folderKey.value = repo.value.folder_id == null ? '' : String(repo.value.folder_id)
}

async function load() {
  if (!repo.value || !repoPathMatches(repo.value, props.repoPath)) {
    return
  }
  dataLoading.value = true
  loadError.value = ''
  message.value = ''
  try {
    syncFormFromRepo()
    const folderRes = await foldersApi.list()
    folderOptions.value = [
      { id: '', label: i18n.global.t('nav.root') },
      ...(folderRes.folders ?? []).map((folder) => ({
        id: String(folder.id),
        label: folder.path || folder.name,
      })),
    ]
  } catch (err) {
    loadError.value = localizeError(err, 'errors.loadSettings')
  } finally {
    dataLoading.value = false
  }
}

async function save() {
  if (!repo.value) {
    return
  }
  saving.value = true
  formError.value = ''
  message.value = ''
  try {
    const updated = await reposApi.update(props.repoPath, {
      description: description.value,
      private: isPrivate.value,
      default_branch: defaultBranch.value,
      protect_default_branch: protectBranch.value,
    })
    const currentFolder = repo.value.folder_id ?? null
    const nextFolder = parseFolderKey(folderKey.value)
    if (currentFolder !== nextFolder) {
      const moved = await reposApi.move(props.repoPath, nextFolder)
      repo.value = moved
      const newPath = moved.folder_path ? `${moved.folder_path}/${moved.name}` : moved.name
      await router.replace({
        name: 'repo-settings',
        params: { repoPath: newPath },
      })
      message.value = i18n.global.t('settings.savedMoved')
      return
    }
    repo.value = updated
    message.value = i18n.global.t('settings.saved')
  } catch (err) {
    formError.value = localizeError(err, 'errors.saveFailed')
  } finally {
    saving.value = false
  }
}

async function remove() {
  if (!confirm(i18n.global.t('settings.confirmDelete', { path: props.repoPath }))) {
    return
  }
  deleting.value = true
  formError.value = ''
  try {
    await reposApi.remove(props.repoPath)
    await router.push({ name: 'home' })
  } catch (err) {
    formError.value = localizeError(err, 'errors.deleteFailed')
    deleting.value = false
  }
}

watch([() => props.repoPath, repo], load, { immediate: true })
</script>

<template>
  <PageState :loading="loading" :error="loadError">
    <form
      v-if="repo && repoPathMatches(repo, repoPath)"
      class="term-block max-w-2xl"
      @submit.prevent="save"
    >
      <div class="term-block-hd">{{ $t('settings.title') }}</div>
      <div class="term-block-bd space-y-4">
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('settings.description') }}</span>
          <input v-model="description" :class="field" />
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('settings.defaultBranch') }}</span>
          <input v-model="defaultBranch" :class="[field, 'font-mono']" :placeholder="$t('modal.defaultBranchPlaceholder')" />
        </label>
        <label class="flex items-center gap-2 text-sm">
          <input v-model="isPrivate" type="checkbox" class="accent-ink" />
          {{ $t('settings.private') }}
        </label>
        <label class="flex items-center gap-2 text-sm">
          <input v-model="protectBranch" type="checkbox" class="accent-ink" />
          {{ $t('settings.protectBranch') }}
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('settings.folder') }}</span>
          <select v-model="folderKey" :class="field">
            <option v-for="opt in folderOptions" :key="opt.id" :value="opt.id">
              {{ opt.id === '' ? $t('nav.root') : opt.label }}
            </option>
          </select>
        </label>
        <p v-if="formError" class="text-sm text-warn">{{ formError }}</p>
        <p v-else-if="message" class="text-sm text-ink">{{ message }}</p>
        <button type="submit" :class="btnPrimary" :disabled="saving">
          {{ saving ? $t('settings.saving') : $t('settings.save') }}
        </button>
      </div>
    </form>

    <div v-if="repo && repoPathMatches(repo, repoPath)" class="term-block mt-6 max-w-2xl">
      <h2 class="term-block-hd text-warn">{{ $t('settings.danger') }}</h2>
      <div class="flex flex-wrap items-center justify-between gap-3 px-3 py-3 sm:px-4">
        <div>
          <p class="font-medium">{{ $t('settings.delete') }}</p>
          <p class="text-sm text-ink-muted">{{ $t('settings.dangerText') }}</p>
        </div>
        <button type="button" :class="btnDanger" :disabled="deleting" @click="remove">
          {{ deleting ? $t('settings.deleting') : $t('settings.delete') }}
        </button>
      </div>
    </div>
  </PageState>
</template>
