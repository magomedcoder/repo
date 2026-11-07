<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { reposApi } from '@/api'
import type { BlobContent, BlameResult } from '@/api/types'
import BlobCodeViewer from '@/components/repo/BlobCodeViewer.vue'
import BlobToolbar from '@/components/repo/BlobToolbar.vue'
import RepoShell from '@/components/repo/RepoShell.vue'
import PageState from '@/components/ui/PageState.vue'
import { useRepo } from '@/composables/useRepo'
import type { PageError } from '@/i18n'
import { i18n, localizeError, pageError } from '@/i18n'

const props = defineProps<{
  repoPath: string
}>()

const { repo, loading: repoLoading, error: repoError } = useRepo(() => props.repoPath)

const route = useRoute()
const blob = ref<BlobContent | null>(null)
const blame = ref<BlameResult | null>(null)
const error = ref<PageError | string>('')
const blameError = ref('')
const loading = ref(true)
const blameLoading = ref(false)
const showBlame = ref(false)

const refName = computed(
  () => (route.query.ref as string) || repo.value?.default_branch || 'main',
)
const filePath = computed(() => (route.query.path as string) || '')
const parentDir = computed(() => {
  const parts = filePath.value.split('/')
  parts.pop()
  return parts.join('/')
})

const rawFilename = computed(() => {
  const parts = filePath.value.split('/')
  return parts[parts.length - 1] || i18n.global.t('blob.fallbackName')
})

const rawHref = computed(() => reposApi.rawHref(props.repoPath, {
  ref: refName.value,
  path: filePath.value
}))

async function loadBlob() {
  loading.value = true
  error.value = ''
  blame.value = null
  try {
    if (!filePath.value) {
      error.value = localizeError(new Error('path'), 'errors.pathRequired')
      return
    }
    blob.value = await reposApi.blob(props.repoPath, {
      ref: refName.value,
      path: filePath.value,
    })
  } catch (err) {
    error.value = pageError(err, 'errors.loadFile')
  } finally {
    loading.value = false
  }
}

async function loadBlame() {
  if (!filePath.value || blob.value?.is_binary) {
    return
  }

  blameLoading.value = true
  blameError.value = ''
  try {
    blame.value = await reposApi.blame(props.repoPath, {
      ref: refName.value,
      path: filePath.value,
    })
  } catch (err) {
    blameError.value = localizeError(err, 'errors.loadBlame')
  } finally {
    blameLoading.value = false
  }
}

async function toggleBlame() {
  showBlame.value = !showBlame.value
  if (showBlame.value && !blame.value && !blameLoading.value) {
    await loadBlame()
  }
}

watch(
  () => [props.repoPath, route.query.ref, route.query.path, repo.value?.default_branch],
  () => {
    showBlame.value = false
    loadBlob()
  },
  { immediate: true },
)
</script>

<template>
  <RepoShell
    :repo-path="repoPath"
    tab="summary"
    :repo="repo"
    :loading="repoLoading"
    :error="repoError"
  >
    <PageState :loading="loading" :error="error">
      <div v-if="blob" class="surface">
        <BlobToolbar
          :repo-path="repoPath"
          :ref-name="refName"
          :file-path="filePath"
          :blob="blob"
          :show-blame="showBlame"
          :parent-dir="parentDir"
          :raw-href="rawHref"
          :raw-filename="rawFilename"
          @toggle-blame="toggleBlame"
        />
        <BlobCodeViewer
          :blob="blob"
          :show-blame="showBlame"
          :blame="blame"
          :blame-loading="blameLoading"
          :blame-error="blameError"
          :filename="rawFilename"
        />
      </div>
    </PageState>
  </RepoShell>
</template>
