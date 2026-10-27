<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { foldersApi } from '@/api'
import type { Folder, RepoSummary } from '@/api/types'
import ContentsList from '@/components/ContentsList.vue'
import { useAuth } from '@/composables/useAuth'
import { localizeError } from '@/i18n'

const props = defineProps<{ id: string }>()

const auth = useAuth()
const folder = ref<Folder | null>(null)
const repos = ref<RepoSummary[]>([])
const error = ref('')
const loading = ref(true)

const folderId = computed(() => {
  const n = Number(props.id)
  return Number.isFinite(n) && n > 0 ? n : NaN
})

async function load() {
  loading.value = true
  error.value = ''
  if (!Number.isFinite(folderId.value)) {
    error.value = localizeError(new Error('invalid'), 'errors.loadFolder')
    loading.value = false
    return
  }
  try {
    const res = await foldersApi.get(folderId.value)
    folder.value = res.folder ?? null
    repos.value = res.repos ?? []
  } catch (err) {
    error.value = localizeError(err, 'errors.loadFolder')
  } finally {
    loading.value = false
  }
}

watch(() => props.id, load, { immediate: true })
</script>

<template>
  <div>
    <p class="mb-3 font-mono text-[14px] font-medium text-ink">{{ folder?.path || $t('folder.fallback') }}</p>

    <p v-if="loading" class="text-sm text-ink-muted">{{ $t('common.loading') }}</p>
    <p v-else-if="error" class="text-sm text-warn">{{ error }}</p>
    <ContentsList
      v-else
      :repos="repos"
      :folder-path="folder?.path || ''"
      :repo-owner="auth.user.value?.username"
    />
  </div>
</template>
