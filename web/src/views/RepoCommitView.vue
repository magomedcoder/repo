<script setup lang="ts">
import { onUnmounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { reposApi } from '@/api'
import type { CommitDiff, CommitInfo } from '@/api/types'
import { useBreadcrumbs } from '@/composables/useBreadcrumbs'
import { ApiError } from '@/api/client'

const props = defineProps<{
  owner: string
  repoPath: string
  sha: string
}>()

const { setBreadcrumbs, clearBreadcrumbs } = useBreadcrumbs()

const commit = ref<CommitInfo | null>(null)
const diff = ref<CommitDiff | null>(null)
const error = ref('')
const loading = ref(true)

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
  crumbs.push({
    label: 'commits',
    to: {
      name: 'repo-commits',
      params: {
        owner: props.owner,
        repoPath: props.repoPath
      }
    },
  })
  crumbs.push({ label: props.sha.slice(0, 7) })
  setBreadcrumbs(crumbs as never)
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [c, d] = await Promise.all([
      reposApi.commit(props.owner, props.repoPath, props.sha),
      reposApi.diff(props.owner, props.repoPath, props.sha),
    ])
    commit.value = c
    diff.value = d
    setCrumbs()
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'Failed to load commit'
  } finally {
    loading.value = false
  }
}

watch(() => [props.owner, props.repoPath, props.sha], load, { immediate: true })
onUnmounted(clearBreadcrumbs)
</script>

<template>
  <div>
    <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
      <h1 class="font-display text-2xl font-bold">Commit</h1>
      <RouterLink
        class="btn-ghost"
        :to="{
          name: 'repo-commits',
          params: {
            owner, repoPath
          }
        }"
      >
        All commits
      </RouterLink>
    </div>

    <p v-if="loading" class="text-sm text-ink-muted">Loading...</p>
    <p v-else-if="error" class="text-sm text-warn">{{ error }}</p>
    <template v-else-if="commit">
      <div class="panel mb-4 p-4">
        <pre class="whitespace-pre-wrap font-sans text-sm">{{ commit.message }}</pre>
        <p class="mt-3 font-mono text-xs text-ink-muted">
          {{ commit.sha }} {{ commit.author_name }} &lt;{{ commit.author_email }}&gt;
          {{ new Date(commit.authored_at).toLocaleString() }}
        </p>
      </div>

      <div v-if="diff" class="space-y-3">
        <h2 class="text-xs font-semibold uppercase tracking-wide text-ink-muted">
          {{ diff.files.length }} files changed
        </h2>
        <article v-for="file in diff.files" :key="file.path + file.status" class="panel overflow-hidden">
          <header class="flex flex-wrap items-center gap-2 border-b border-line px-4 py-2 text-sm">
            <span class="rounded bg-paper-2 px-1.5 py-0.5 text-xs font-semibold uppercase">{{ file.status }}</span>
            <span class="font-mono">{{ file.path }}</span>
            <span v-if="file.old_path" class="font-mono text-ink-muted">from {{ file.old_path }}</span>
          </header>
          <pre class="overflow-x-auto p-4 font-mono text-xs leading-relaxed">{{ file.patch }}</pre>
        </article>
      </div>
    </template>
  </div>
</template>
