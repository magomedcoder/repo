<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { reposApi } from '@/api'
import type { BlobContent } from '@/api/types'
import { useBreadcrumbs } from '@/composables/useBreadcrumbs'
import { localizeError } from '@/i18n'

const props = defineProps<{
  owner: string
  repoPath: string
}>()

const route = useRoute()
const { setBreadcrumbs, clearBreadcrumbs } = useBreadcrumbs()

const blob = ref<BlobContent | null>(null)
const error = ref('')
const loading = ref(true)

const refName = computed(() => (route.query.ref as string) || 'main')
const filePath = computed(() => (route.query.path as string) || '')

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
  filePath.value.split('/').forEach((p) => {
    if (p) {
      crumbs.push({ label: p })
    }
  })
  setBreadcrumbs(crumbs as never)
}

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
    setCrumbs()
  } catch (err) {
    error.value = localizeError(err, 'errors.loadFile')
  } finally {
    loading.value = false
  }
}

const parentDir = computed(() => {
  const parts = filePath.value.split('/')
  parts.pop()
  return parts.join('/')
})

watch(() => [props.owner, props.repoPath, route.query.ref, route.query.path], load, { immediate: true })
onUnmounted(clearBreadcrumbs)
</script>

<template>
  <div>
    <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="font-display text-2xl font-bold">{{ filePath }}</h1>
        <p class="font-mono text-sm text-ink-muted">{{ refName }}</p>
      </div>
      <div class="flex gap-2">
        <RouterLink
          class="btn-ghost"
          :to="{
            name: 'repo-tree',
            params: { owner, repoPath },
            query: {
              ref: refName,
              path: parentDir || undefined
            },
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

    <p v-if="loading" class="text-sm text-ink-muted">{{ $t('common.loading') }}</p>
    <p v-else-if="error" class="text-sm text-warn">{{ error }}</p>
    <div v-else-if="blob" class="panel overflow-hidden">
      <div class="flex items-center justify-between border-b border-line px-4 py-2 text-xs text-ink-muted">
        <span>{{ $t('common.bytes', { n: blob.size }) }}</span>
        <span>{{ blob.is_binary ? $t('common.binary') : blob.encoding }}</span>
      </div>
      <pre
        v-if="!blob.is_binary"
        class="overflow-x-auto p-4 font-mono text-sm leading-relaxed"
      >{{ blob.content }}</pre>
      <p v-else class="p-4 text-sm text-ink-muted">
        {{ $t('blob.binary') }}
      </p>
    </div>
  </div>
</template>
