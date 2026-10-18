<script setup lang="ts">
import { useRepoOwner } from '@/composables/useRepoOwner'
import { btnDanger, btnPrimary, field } from '@/lib/classes'
import { inject, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { foldersApi, reposApi } from '@/api'
import type { FolderTreeNode, Repository } from '@/api/types'
import RepoShell from '@/components/repo/RepoShell.vue'
import PageState from '@/components/ui/PageState.vue'
import { i18n, localizeError } from '@/i18n'

const owner = useRepoOwner()

const props = defineProps<{
  repoPath: string
}>()

const router = useRouter()
const reloadSidebar = inject<() => void>('reloadSidebar', () => {})

const repo = ref<Repository | null>(null)
const description = ref('')
const isPrivate = ref(false)
const protectBranch = ref(true)
const defaultBranch = ref('main')
const folderKey = ref('')
const folderOptions = ref<{
  id: string
  label: string
}[]>([{
  id: '',
  label: 'Root'
}])
const loadError = ref('')
const formError = ref('')
const message = ref('')
const loading = ref(true)
const saving = ref(false)
const deleting = ref(false)

function parseFolderKey(key: string): number | null {
  return key === '' ? null : Number(key)
}

function flattenFolders(nodes: FolderTreeNode[], acc: { id: number; label: string }[] = []) {
  for (const node of nodes) {
    acc.push({
      id: node.id,
      label: node.path
    })
    if (node.children?.length) {
      flattenFolders(node.children, acc)
    }
  }
  return acc
}

async function load() {
  loading.value = true
  loadError.value = ''
  message.value = ''
  try {
    repo.value = await reposApi.get(owner.value, props.repoPath)
    description.value = repo.value.description
    isPrivate.value = repo.value.is_private
    protectBranch.value = repo.value.protect_default_branch !== false
    defaultBranch.value = repo.value.default_branch
    folderKey.value = repo.value.folder_id == null ? '' : String(repo.value.folder_id)
    const tree = await foldersApi.tree()
    folderOptions.value = [
      {
        id: '',
        label: 'Root'
      },
      ...flattenFolders(tree.tree ?? []).map((folder) => ({
        id: String(folder.id),
        label: folder.label
      })),
    ]
  } catch (err) {
    loadError.value = localizeError(err, 'errors.loadSettings')
  } finally {
    loading.value = false
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
    const updated = await reposApi.update(owner.value, props.repoPath, {
      description: description.value,
      private: isPrivate.value,
      default_branch: defaultBranch.value,
      protect_default_branch: protectBranch.value,
    })
    const currentFolder = repo.value.folder_id ?? null
    const nextFolder = parseFolderKey(folderKey.value)
    if (currentFolder !== nextFolder) {
      const moved = await reposApi.move(owner.value, props.repoPath, nextFolder)
      repo.value = moved
      const newPath = moved.folder_path ? `${moved.folder_path}/${moved.name}` : moved.name
      reloadSidebar()
      await router.replace({
        name: 'repo-settings',
        params: {
            repoPath: newPath
        }
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
    await reposApi.remove(owner.value, props.repoPath)
    reloadSidebar()
    await router.push({ name: 'home' })
  } catch (err) {
    formError.value = localizeError(err, 'errors.deleteFailed')
    deleting.value = false
  }
}

watch(() => [owner.value, props.repoPath], load, { immediate: true })
</script>

<template>
  <RepoShell
    :owner="owner"
    :repo-path="repoPath"
    tab="settings"
  >
    <PageState :loading="loading" :error="loadError">
      <form
        v-if="repo" class="max-w-2xl space-y-4 rounded-md border border-line bg-white p-4"
        @submit.prevent="save"
      >
        <h1 class="text-xl font-semibold">{{ $t('settings.title') }}</h1>
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('settings.description') }}</span>
          <input v-model="description" :class="field" />
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('settings.defaultBranch') }}</span>
          <input v-model="defaultBranch" :class="[field, 'font-mono']" />
        </label>
        <label class="flex items-center gap-2 text-sm">
          <input
            v-model="isPrivate"
            type="checkbox"
            class="accent-[#1f883d]"
          />
          {{ $t('settings.private') }}
        </label>
        <label class="flex items-center gap-2 text-sm">
          <input v-model="protectBranch" type="checkbox" />
          {{ $t('settings.protectBranch') }}
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('settings.folder') }}</span>
          <select v-model="folderKey" :class="field">
            <option
              v-for="opt in folderOptions"
              :key="opt.id"
              :value="opt.id"
            >
              {{ opt.id === '' ? $t('nav.root') : opt.label }}
            </option>
          </select>
        </label>
        <p v-if="formError" class="text-sm text-warn">{{ formError }}</p>
        <p v-else-if="message" class="text-sm text-moss">{{ message }}</p>
        <button type="submit" :class="btnPrimary" :disabled="saving">
          {{ saving ? $t('settings.saving') : $t('settings.save') }}
        </button>
      </form>

      <div v-if="repo" class="mt-6 max-w-2xl overflow-hidden rounded-md border border-warn">
        <h2 class="border-b border-warn px-4 py-2 text-sm font-semibold">{{ $t('settings.danger') }}</h2>
        <div class="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
          <div>
            <p class="font-semibold">{{ $t('settings.delete') }}</p>
            <p class="text-sm text-ink-muted">{{ $t('settings.dangerText') }}</p>
          </div>
          <button
            type="button"
            :class="btnDanger"
            :disabled="deleting"
            @click="remove"
          >
            {{ deleting ? $t('settings.deleting') : $t('settings.delete') }}
          </button>
        </div>
      </div>
    </PageState>
  </RepoShell>
</template>
