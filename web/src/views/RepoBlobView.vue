<script setup lang="ts">
import { btnGhost, dataTable } from '@/lib/classes'
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { reposApi } from '@/api'
import type { BlobContent, BlameResult } from '@/api/types'
import RepoShell from '@/components/repo/RepoShell.vue'
import PageState from '@/components/ui/PageState.vue'
import { localizeError, pageError } from '@/i18n'

const props = defineProps<{
  owner: string
  repoPath: string
}>()

const route = useRoute()
const blob = ref<BlobContent | null>(null)
const blame = ref<BlameResult | null>(null)
const error = ref('')
const blameError = ref('')
const loading = ref(true)
const blameLoading = ref(false)
const showBlame = ref(false)

const refName = computed(() => (route.query.ref as string) || 'main')
const filePath = computed(() => (route.query.path as string) || '')
const parentDir = computed(() => {
  const parts = filePath.value.split('/')
  parts.pop()
  return parts.join('/')
})

const rawFilename = computed(() => {
  const parts = filePath.value.split('/')
  return parts[parts.length - 1] || 'file'
})

const rawHref = computed(() => `/api/repos/${props.owner}/${props.repoPath}/raw?ref=${encodeURIComponent(refName.value)}&path=${encodeURIComponent(filePath.value)}`)

function lineClass(line: string, filename: string) {
  const ext = filename.split('.').pop()?.toLowerCase() || ''
  const comment = ext === 'py' || ext === 'sh' ? /^\s*#/ : /^\s*(\/\/|\/\*|\*)/
  if (comment.test(line)) {
    return 'text-ink-muted'
  }

  if (/^\s*(func|class|def|import|package|type|const|let|var|return)\b/.test(line)) {
    return 'text-accent'
  }

  return ''
}

function shortSha(sha: string) {
  return sha.slice(0, 7)
}

async function loadBlob() {
  loading.value = true
  error.value = ''
  blame.value = null
  try {
    if (!filePath.value) {
      error.value = localizeError(new Error('path'), 'errors.pathRequired')
      return
    }
    blob.value = await reposApi.blob(props.owner, props.repoPath, {
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
    blame.value = await reposApi.blame(props.owner, props.repoPath, {
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
  () => [props.owner, props.repoPath, route.query.ref, route.query.path],
  () => {
    showBlame.value = false
    loadBlob()
  },
  { immediate: true },
)
</script>

<template>
  <RepoShell
    :owner="owner"
    :repo-path="repoPath"
    tab="code"
  >
    <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
      <p class="min-w-0 truncate font-mono text-sm">
        <span class="text-ink-muted">{{ refName }} /</span> {{ filePath }}
      </p>
      <div class="flex gap-2">
        <button
          v-if="blob && !blob.is_binary"
          type="button"
          :class="btnGhost"
          :class="{ 'bg-paper-2': showBlame }"
          @click="toggleBlame"
        >
          {{ showBlame ? $t('blob.hideBlame') : $t('blob.blame') }}
        </button>
        <RouterLink
          :class="btnGhost"
          :to="{
            name: 'repo-tree',
            params: { repoPath },
            query: {
              ref: refName,
              path: parentDir || undefined
            }
          }"
        >
          {{ $t('blob.tree') }}
        </RouterLink>
        <a
          :class="btnGhost"
          :href="rawHref"
          :download="rawFilename"
          target="_blank"
          rel="noreferrer"
        >
          {{ $t('blob.raw') }}
        </a>
      </div>
    </div>

    <PageState :loading="loading" :error="error">
      <div v-if="blob && showBlame && !blob.is_binary" class="overflow-hidden rounded-md border border-line bg-white">
        <p v-if="blameLoading" class="p-4 text-sm text-ink-muted">{{ $t('common.loading') }}</p>
        <p v-else-if="blameError" class="p-4 text-sm text-warn">{{ blameError }}</p>
        <div v-else-if="blame" class="overflow-x-auto">
          <table class="w-full font-mono text-xs">
            <tbody>
              <tr
                v-for="line in blame.lines"
                :key="line.line"
                class="border-b border-line/60 hover:bg-paper"
              >
                <td class="whitespace-nowrap px-2 py-0.5 text-right text-ink-muted">{{ line.line }}</td>
                <td class="whitespace-nowrap px-2 py-0.5 text-ink-muted">{{ shortSha(line.commit_sha) }}</td>
                <td class="whitespace-nowrap px-2 py-0.5 text-ink-muted">{{ line.author_name }}</td>
                <td class="px-2 py-0.5">{{ line.content }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
      <div v-else-if="blob" class="overflow-hidden rounded-md border border-line bg-white">
        <div class="flex items-center justify-between border-b border-line bg-paper px-4 py-2 text-xs text-ink-muted">
          <span>{{ $t('common.bytes', { n: blob.size }) }}</span>
          <span>{{ blob.is_binary ? $t('common.binary') : blob.encoding }}</span>
        </div>
        <table v-if="!blob.is_binary" :class="[dataTable, 'font-mono']">
          <tbody>
            <tr v-for="(line, i) in (blob.content || '').split('\n')" :key="i">
              <td class="w-10 text-right text-ink-muted">{{ i + 1 }}</td>
              <td :class="lineClass(line, rawFilename)">{{ line || ' ' }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else class="p-4 text-sm text-ink-muted">{{ $t('blob.binary') }}</p>
      </div>
    </PageState>
  </RepoShell>
</template>
