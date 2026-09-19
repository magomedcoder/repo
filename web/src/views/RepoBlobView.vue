<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { reposApi } from '@/api'
import type { BlobContent } from '@/api/types'
import RepoShell from '@/components/repo/RepoShell.vue'
import PageState from '@/components/ui/PageState.vue'
import { localizeError } from '@/i18n'

const props = defineProps<{
  owner: string
  repoPath: string
}>()

const route = useRoute()
const blob = ref<BlobContent | null>(null)
const error = ref('')
const loading = ref(true)

const refName = computed(() => (route.query.ref as string) || 'main')
const filePath = computed(() => (route.query.path as string) || '')
const parentDir = computed(() => {
  const parts = filePath.value.split('/')
  parts.pop()
  return parts.join('/')
})

async function load() {
  loading.value = true
  error.value = ''
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
    error.value = localizeError(err, 'errors.loadFile')
  } finally {
    loading.value = false
  }
}

watch(() => [props.owner, props.repoPath, route.query.ref, route.query.path], load, { immediate: true })
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
        <RouterLink
          class="btn-ghost"
          :to="{
            name: 'repo-tree',
            params: { owner, repoPath },
            query: {
              ref: refName,
              path: parentDir || undefined
            }
          }"
        >
          {{ $t('blob.tree') }}
        </RouterLink>
        <a
          class="btn-ghost"
          :href="`/api/repos/${owner}/${repoPath}/raw?ref=${encodeURIComponent(refName)}&path=${encodeURIComponent(filePath)}`"
          target="_blank"
          rel="noreferrer"
        >
          {{ $t('blob.raw') }}
        </a>
      </div>
    </div>

    <PageState :loading="loading" :error="error">
      <div v-if="blob" class="overflow-hidden rounded-md border border-line bg-white">
        <div class="flex items-center justify-between border-b border-line bg-paper px-4 py-2 text-xs text-ink-muted">
          <span>{{ $t('common.bytes', { n: blob.size }) }}</span>
          <span>{{ blob.is_binary ? $t('common.binary') : blob.encoding }}</span>
        </div>
        <pre v-if="!blob.is_binary" class="overflow-x-auto p-4 font-mono text-xs leading-5">{{ blob.content }}</pre>
        <p v-else class="p-4 text-sm text-ink-muted">{{ $t('blob.binary') }}</p>
      </div>
    </PageState>
  </RepoShell>
</template>
