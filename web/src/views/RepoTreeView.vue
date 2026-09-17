<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { reposApi } from '@/api'
import type { TreeEntry } from '@/api/types'
import { useBreadcrumbs } from '@/composables/useBreadcrumbs'
import { ApiError } from '@/api/client'

const props = defineProps<{
  owner: string
  repoPath: string
}>()

const route = useRoute()
const { setBreadcrumbs, clearBreadcrumbs } = useBreadcrumbs()

const tree = ref<TreeEntry[]>([])
const error = ref('')
const loading = ref(true)

const refName = computed(() => (route.query.ref as string) || 'main')
const dirPath = computed(() => (route.query.path as string) || '')

function setCrumbs() {
  const crumbs: { label: string; to?: object | string }[] = [
    {
      label: props.owner,
      to: '/'
    },
  ]
  const parts = props.repoPath.split('/')
  parts.forEach((part, i) => {
    const isLast = i === parts.length - 1
    crumbs.push(isLast
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
  if (dirPath.value) {
    dirPath.value.split('/').forEach((p) => crumbs.push({ label: p }))
  }

  setBreadcrumbs(crumbs as never)
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await reposApi.tree(props.owner, props.repoPath, {
      ref: refName.value,
      path: dirPath.value,
    })
    tree.value = res.tree ?? []
    setCrumbs()
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'Failed to load tree'
  } finally {
    loading.value = false
  }
}

function parentPath() {
  if (!dirPath.value) {
    return null
  }
  const parts = dirPath.value.split('/')
  parts.pop()
  return parts.join('/')
}

function entryLink(entry: TreeEntry) {
  if (entry.type === 'tree') {
    return {
      name: 'repo-tree' as const,
      params: {
        owner: props.owner,
        repoPath: props.repoPath
      },
      query: {
        ref: refName.value,
        path: entry.path
      },
    }
  }
  return {
    name: 'repo-blob' as const,
    params: {
      owner: props.owner,
      repoPath: props.repoPath
    },
    query: {
      ref: refName.value,
      path: entry.path
    },
  }
}

watch(() => [props.owner, props.repoPath, route.query.ref, route.query.path], load, { immediate: true })
onUnmounted(clearBreadcrumbs)
</script>

<template>
  <div>
    <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="font-display text-2xl font-bold">Files</h1>
        <p class="font-mono text-sm text-ink-muted">
          {{ refName }}<span v-if="dirPath"> / {{ dirPath }}</span>
        </p>
      </div>
      <RouterLink
        class="btn-ghost"
        :to="{
          name: 'repo',
          params: { owner, repoPath }
        }">Repo home</RouterLink>
    </div>

    <div v-if="dirPath" class="mb-3">
      <RouterLink
        class="text-sm text-moss hover:underline"
        :to="{
          name: 'repo-tree',
          params: { owner, repoPath },
          query: {
            ref: refName,
            path: parentPath() || undefined
          },
        }"
      >
        <- Parent directory
      </RouterLink>
    </div>

    <p v-if="loading" class="text-sm text-ink-muted">Loading...</p>
    <p v-else-if="error" class="text-sm text-warn">{{ error }}</p>
    <ul v-else class="divide-y divide-line overflow-hidden rounded-lg border border-line bg-white/80">
      <li v-for="entry in tree" :key="entry.path">
        <RouterLink :to="entryLink(entry)" class="flex items-center gap-3 px-4 py-2.5 hover:bg-moss-soft/40">
          <span class="w-10 font-mono text-xs text-ink-muted">{{ entry.type }}</span>
          <span class="flex-1 font-medium">{{ entry.name }}</span>
          <span v-if="entry.type === 'blob'" class="font-mono text-xs text-ink-muted">{{ entry.size }} B</span>
        </RouterLink>
      </li>
    </ul>
  </div>
</template>
