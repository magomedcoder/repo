<script setup lang="ts">
import { inject, onUnmounted, ref, watch } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { foldersApi, reposApi } from '@/api'
import type { FolderTreeNode, Repository } from '@/api/types'
import { useBreadcrumbs } from '@/composables/useBreadcrumbs'
import { ApiError } from '@/api/client'

const props = defineProps<{
  owner: string
  repoPath: string
}>()

const router = useRouter()
const { setBreadcrumbs, clearBreadcrumbs } = useBreadcrumbs()
const reloadSidebar = inject<() => void>('reloadSidebar', () => {})

const repo = ref<Repository | null>(null)
const description = ref('')
const isPrivate = ref(false)
const defaultBranch = ref('main')
const folderKey = ref('')
const folderOptions = ref<{
  id: string;
  label: string
}[]>([{
  id: '',
  label: 'Root'
}])
const error = ref('')
const message = ref('')
const loading = ref(true)
const saving = ref(false)
const deleting = ref(false)

function parseFolderKey(key: string): number | null {
  return key === '' ? null : Number(key)
}

function setCrumbs() {
  const crumbs: { label: string; to?: object | string }[] = [{ label: props.owner, to: '/' }]
  props.repoPath.split('/').forEach((part, i, arr) => {
    crumbs.push(i === arr.length - 1
      ? {
        label: part,
        to: {
          name: 'repo',
          params: {
            owner: props.owner,
            repoPath: props.repoPath
          }
        }
      }
      : { label: part })
  })
  crumbs.push({ label: 'settings' })
  setBreadcrumbs(crumbs as never)
}

function flattenFolders(nodes: FolderTreeNode[], acc: { id: number; label: string }[] = []) {
  for (const n of nodes) {
    acc.push({ id: n.id, label: n.path })
    if (n.children?.length) {
      flattenFolders(n.children, acc)
    }
  }
  return acc
}

async function load() {
  loading.value = true
  error.value = ''
  message.value = ''
  try {
    repo.value = await reposApi.get(props.owner, props.repoPath)
    description.value = repo.value.description
    isPrivate.value = repo.value.is_private
    defaultBranch.value = repo.value.default_branch
    folderKey.value = repo.value.folder_id == null ? '' : String(repo.value.folder_id)
    setCrumbs()

    const tree = await foldersApi.tree()
    folderOptions.value = [
      {
        id: '',
        label: 'Root'
      },
      ...flattenFolders(tree.tree ?? []).map((f) => ({
        id: String(f.id),
        label: f.label
      })),
    ]
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'Failed to load settings'
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!repo.value) return
  saving.value = true
  error.value = ''
  message.value = ''
  try {
    const updated = await reposApi.update(props.owner, props.repoPath, {
      description: description.value,
      private: isPrivate.value,
      default_branch: defaultBranch.value,
    })

    const currentFolder = repo.value.folder_id ?? null
    const nextFolder = parseFolderKey(folderKey.value)
    if (currentFolder !== nextFolder) {
      const moved = await reposApi.move(props.owner, props.repoPath, nextFolder)
      repo.value = moved
      const newPath = moved.folder_path ? `${moved.folder_path}/${moved.name}` : moved.name
      reloadSidebar()
      await router.replace({
        name: 'repo-settings',
        params: {
          owner: props.owner,
          repoPath: newPath
        },
      })
      message.value = 'Saved and moved'
      return
    }

    repo.value = updated
    message.value = 'Saved'
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'Save failed'
  } finally {
    saving.value = false
  }
}

async function remove() {
  if (!confirm(`Delete repository ${props.repoPath}? This cannot be undone.`)) return
  deleting.value = true
  error.value = ''
  try {
    await reposApi.remove(props.owner, props.repoPath)
    reloadSidebar()
    await router.push({ name: 'home' })
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'Delete failed'
    deleting.value = false
  }
}

watch(() => [props.owner, props.repoPath], load, { immediate: true })
onUnmounted(clearBreadcrumbs)
</script>

<template>
  <div>
    <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
      <h1 class="font-display text-2xl font-bold">Settings</h1>
      <RouterLink
        class="btn-ghost"
        :to="{
          name: 'repo',
          params: { owner, repoPath }
        }">Repo home</RouterLink>
    </div>

    <p v-if="loading" class="text-sm text-ink-muted">Loading...</p>
    <form v-else class="panel max-w-xl space-y-4 p-5" @submit.prevent="save">
      <label class="block text-sm">
        <span class="mb-1 block font-medium">Description</span>
        <input v-model="description" class="input" />
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-medium">Default branch</span>
        <input v-model="defaultBranch" class="input font-mono" />
      </label>
      <label class="flex items-center gap-2 text-sm">
        <input v-model="isPrivate" type="checkbox" class="accent-moss" />
        Private repository
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-medium">Folder</span>
        <select v-model="folderKey" class="input">
          <option v-for="opt in folderOptions" :key="opt.id" :value="opt.id">
            {{ opt.label }}
          </option>
        </select>
      </label>

      <p v-if="error" class="text-sm text-warn">{{ error }}</p>
      <p v-else-if="message" class="text-sm text-moss">{{ message }}</p>

      <div class="flex flex-wrap gap-2 pt-2">
        <button type="submit" class="btn-primary" :disabled="saving">
          {{ saving ? 'Saving...' : 'Save changes' }}
        </button>
      </div>
    </form>

    <div class="panel mt-8 max-w-xl border-warn/30 p-5">
      <h2 class="font-display text-lg font-bold text-warn">Danger zone</h2>
      <p class="mt-1 text-sm text-ink-muted">Permanently delete this repository and its git data.</p>
      <button type="button" class="btn-danger mt-4" :disabled="deleting" @click="remove">
        {{ deleting ? 'Deleting...' : 'Delete repository' }}
      </button>
    </div>
  </div>
</template>
