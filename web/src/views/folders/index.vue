<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { foldersApi } from '@/api'
import type { Folder } from '@/api/types'
import { useAuth } from '@/composables/useAuth'
import { folderKey, folderLocation } from '@/lib/folderNav'
import { idleAge } from '@/lib/idle'
import { i18n, localizeError } from '@/i18n'
import Icon from '@/components/ui/Icon.vue'
import GlassTable from '@/components/ui/GlassTable.vue'

const auth = useAuth()
const folders = ref<Folder[]>([])
const error = ref('')
const loading = ref(true)
const busyPath = ref('')

async function load() {
  if (!auth.isAuthenticated.value) {
    folders.value = []
    loading.value = false
    error.value = ''
    return
  }

  loading.value = true
  error.value = ''
  try {
    const res = await foldersApi.list()
    folders.value = (res.folders ?? []).slice().sort((a, b) => a.path.localeCompare(b.path))
  } catch (err) {
    error.value = localizeError(err, 'errors.loadFailed')
  } finally {
    loading.value = false
  }
}

async function rename(folder: Folder) {
  const next = window.prompt(i18n.global.t('folder.rename'), folder.name)
  if (next == null) {
    return
  }

  const name = next.trim()
  if (!name || name === folder.name) {
    return
  }

  busyPath.value = folder.path
  error.value = ''
  try {
    await foldersApi.rename(folderKey(folder), { name })
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.saveFailed')
  } finally {
    busyPath.value = ''
  }
}

async function remove(folder: Folder) {
  if (!window.confirm(i18n.global.t('folder.confirmDelete', { name: folder.name }))) {
    return
  }
  
  busyPath.value = folder.path
  error.value = ''
  try {
    await foldersApi.remove(folderKey(folder))
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.deleteFailed')
  } finally {
    busyPath.value = ''
  }
}

watch(
  () => auth.isAuthenticated.value,
  () => {
    load()
  },
)

onMounted(() => load())
</script>

<template>
  <div>
    <div class="glass mb-3 px-3 py-1.5 font-mono text-[90%] text-ink sm:px-4">
      <RouterLink to="/" class="text-ink-muted no-underline hover:underline">{{ $t('nav.root') }}</RouterLink>
      <span> / </span>
      <span class="font-bold text-ink">{{ $t('nav.folders') }}</span>
    </div>

    <p v-if="loading" class="text-ink-muted">{{ $t('common.loading') }}</p>
    <p v-else-if="error && !folders.length" class="text-warn">{{ error }}</p>
    <template v-else>
    <p v-if="error" class="mb-2 text-warn">{{ error }}</p>
    <GlassTable>
      <thead>
        <tr>
          <th class="w-[36%]">{{ $t('index.name') }}</th>
          <th>{{ $t('index.path') }}</th>
          <th class="w-[16%]">{{ $t('index.idle') }}</th>
          <th class="w-[18%]" />
        </tr>
      </thead>
      <tbody>
        <tr v-for="folder in folders" :key="folder.path">
          <td class="truncate">
            <RouterLink :to="folderLocation(folder)" class="inline-flex items-center gap-1.5">
              <Icon name="folder" size="sm" class="text-ink-muted" />
              {{ folder.name }}
            </RouterLink>
          </td>
          <td class="truncate font-mono text-[90%] text-ink-muted">{{ folder.path }}</td>
          <td class="whitespace-nowrap">{{ idleAge(folder.updated_at) }}</td>
          <td class="whitespace-nowrap">
            <button
              type="button"
              class="inline-flex items-center gap-1 cursor-pointer border-0 bg-transparent p-0 text-[10pt] font-medium text-ink hover:underline"
              :disabled="busyPath === folder.path"
              @click="rename(folder)"
            >
              <Icon name="pencil" size="sm" />
              {{ $t('folder.rename') }}
            </button>
            <button
              type="button"
              class="ml-2 inline-flex items-center gap-1 cursor-pointer border-0 bg-transparent p-0 text-[10pt] font-medium text-warn hover:underline"
              :disabled="busyPath === folder.path"
              @click="remove(folder)"
            >
              <Icon name="trash" size="sm" />
              {{ $t('folder.delete') }}
            </button>
          </td>
        </tr>
        <tr v-if="!folders.length">
          <td colspan="4">{{ $t('list.noFolders') }}</td>
        </tr>
      </tbody>
    </GlassTable>
    </template>
  </div>
</template>
