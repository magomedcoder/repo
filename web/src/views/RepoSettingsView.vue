<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { foldersApi, reposApi } from '@/api'
import type { RefInfo } from '@/api/types'
import RepoShell from '@/components/repo/RepoShell.vue'
import PageState from '@/components/ui/PageState.vue'
import { useRepo } from '@/composables/useRepo'
import type { PageError } from '@/i18n'
import { i18n, localizeError } from '@/i18n'
import { folderKey } from '@/lib/folderNav'
import { normalizeRepoPath } from '@/lib/repoPath'
import { repoLocation } from '@/lib/repoNav'

const props = defineProps<{
  repoPath: string
}>()

const router = useRouter()
const { repo, loading: repoLoading, error: repoError, reload } = useRepo(() => props.repoPath)

const name = ref('')
const description = ref('')
const isPrivate = ref(false)
const showOnHome = ref(false)
const protectBranch = ref(true)
const defaultBranch = ref('main')
const branches = ref<RefInfo[]>([])
const folderPathKey = ref('')
const folderOptions = ref<{
  path: string
  label: string
}[]>([{
  path: '',
  label: i18n.global.t('nav.root'),
}])
const loadError = ref<PageError | string>('')
const formError = ref('')
const message = ref('')
const dataLoading = ref(false)
const saving = ref(false)
const deleting = ref(false)

const loading = computed(() => dataLoading.value)

const branchOptions = computed(() => {
  const names = branches.value.map((b) => b.name)
  if (defaultBranch.value && !names.includes(defaultBranch.value)) {
    return [defaultBranch.value, ...names]
  }
  
  return names
})

function repoFullPath(folder: string, repoName: string) {
  const f = normalizeRepoPath(folder || '')
  return f ? `${f}/${repoName}` : repoName
}

function syncFormFromRepo() {
  if (!repo.value) {
    return
  }

  name.value = repo.value.name
  description.value = repo.value.description
  isPrivate.value = repo.value.is_private
  showOnHome.value = !!repo.value.show_on_home && !repo.value.is_private
  protectBranch.value = repo.value.protect_default_branch !== false
  defaultBranch.value = repo.value.default_branch
  folderPathKey.value = normalizeRepoPath(repo.value.folder_path || '')
}

async function load() {
  if (!repo.value) {
    return
  }

  dataLoading.value = true
  loadError.value = ''
  message.value = ''
  try {
    syncFormFromRepo()
    const [folderRes, branchRes] = await Promise.all([
      foldersApi.list(),
      reposApi.branches(props.repoPath),
    ])
    folderOptions.value = [
      {
        path: '',
        label: i18n.global.t('nav.root'),
      },
      ...(folderRes.folders ?? []).map((folder) => ({
        path: folderKey(folder),
        label: folder.path || folder.name,
      })),
    ]
    branches.value = branchRes.branches ?? []
    if (branches.value.length && !branches.value.some((b) => b.name === defaultBranch.value)) {
      defaultBranch.value = branches.value[0]?.name || defaultBranch.value
    }
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
    const nextName = name.value.trim()
    const updated = await reposApi.update(props.repoPath, {
      name: nextName,
      description: description.value,
      private: isPrivate.value,
      show_on_home: !isPrivate.value && showOnHome.value,
      default_branch: defaultBranch.value,
      protect_default_branch: protectBranch.value,
    })

    let currentPath = repoFullPath(updated.folder_path || '', updated.name)
    const currentFolder = normalizeRepoPath(updated.folder_path || '')
    const nextFolder = normalizeRepoPath(folderPathKey.value)

    if (currentFolder !== nextFolder) {
      const moved = await reposApi.move(currentPath, nextFolder)
      currentPath = repoFullPath(moved.folder_path || '', moved.name)
      await router.replace(repoLocation('repo-settings', currentPath))
      message.value = i18n.global.t('settings.savedMoved')
      return
    }

    if (currentPath !== props.repoPath) {
      await router.replace(repoLocation('repo-settings', currentPath))
      message.value = i18n.global.t('settings.saved')
      return
    }

    message.value = i18n.global.t('settings.saved')
    await reload()
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
  <RepoShell
    :repo-path="repoPath"
    tab="settings"
    :repo="repo"
    :loading="repoLoading"
    :error="repoError"
  >
    <PageState :loading="loading" :error="loadError">
      <form class="overflow-hidden border border-line bg-paper" @submit.prevent="save">
        <div class="flex items-center gap-2 border-b border-line bg-path px-3 py-1.5 text-[10pt] text-ink">{{ $t('settings.title') }}</div>
        <div class="p-3 sm:p-4 space-y-4">
          <label class="block text-sm">
            <span class="mb-1 block font-semibold">{{ $t('settings.name') }}</span>
            <input
              v-model="name"
              :class="['w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none', 'font-mono']"
              required
              :placeholder="$t('modal.repoNamePlaceholder')"
            />
          </label>
          <label class="block text-sm">
            <span class="mb-1 block font-semibold">{{ $t('settings.description') }}</span>
            <textarea v-model="description" rows="3" :class="['w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none', 'min-h-20 resize-y']" />
          </label>
          <label class="block text-sm">
            <span class="mb-1 block font-semibold">{{ $t('settings.defaultBranch') }}</span>
            <select
              v-model="defaultBranch"
              :class="['w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none', 'font-mono']"
              :disabled="!branchOptions.length"
            >
              <option v-for="b in branchOptions" :key="b" :value="b">{{ b }}</option>
            </select>
            <p v-if="!branchOptions.length" class="mt-1 text-xs text-ink-muted">
              {{ $t('settings.noBranches') }}
            </p>
          </label>
          <label class="flex items-center gap-2 text-sm">
            <input v-model="isPrivate" type="checkbox" class="accent-ink" @change="isPrivate && (showOnHome = false)" />
            {{ $t('settings.private') }}
          </label>
          <label class="flex items-center gap-2 text-sm" :class="isPrivate && 'opacity-50'">
            <input v-model="showOnHome" type="checkbox" class="accent-ink" :disabled="isPrivate" />
            {{ $t('settings.showOnHome') }}
          </label>
          <p class="text-xs text-ink-muted">{{ $t('settings.showOnHomeHint') }}</p>
          <label class="flex items-center gap-2 text-sm">
            <input v-model="protectBranch" type="checkbox" class="accent-ink" />
            {{ $t('settings.protectBranch') }}
          </label>
          <label class="block text-sm">
            <span class="mb-1 block font-semibold">{{ $t('settings.folder') }}</span>
            <select v-model="folderPathKey" class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none">
              <option v-for="opt in folderOptions" :key="opt.path || 'root'" :value="opt.path">
                {{ opt.path === '' ? $t('nav.root') : opt.label }}
              </option>
            </select>
          </label>
          <p v-if="formError" class="text-sm text-warn">{{ formError }}</p>
          <p v-else-if="message" class="text-sm text-ink">{{ message }}</p>
          <button type="submit" class="inline-flex items-center justify-center border border-ink bg-ink px-2.5 py-1 text-[10pt] text-paper hover:bg-ink-muted disabled:cursor-not-allowed disabled:opacity-50" :disabled="saving">
            {{ saving ? $t('settings.saving') : $t('settings.save') }}
          </button>
        </div>
      </form>

      <div class="overflow-hidden border border-line bg-paper mt-6">
        <h2 class="flex items-center gap-2 border-b border-line bg-path px-3 py-1.5 text-[10pt] text-ink text-warn">{{ $t('settings.danger') }}</h2>
        <div class="flex flex-wrap items-center justify-between gap-3 px-3 py-3 sm:px-4">
          <div>
            <p class="font-medium">{{ $t('settings.delete') }}</p>
            <p class="text-sm text-ink-muted">{{ $t('settings.dangerText') }}</p>
          </div>
          <button type="button" class="inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-warn hover:bg-warn-soft disabled:cursor-not-allowed disabled:opacity-50" :disabled="deleting" @click="remove">
            {{ deleting ? $t('settings.deleting') : $t('settings.delete') }}
          </button>
        </div>
      </div>
    </PageState>
  </RepoShell>
</template>
