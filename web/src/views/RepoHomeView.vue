<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { marked } from 'marked'
import { cloneUrl, reposApi } from '@/api'
import type { BlobContent, RefInfo, Repository, RepoStats, TreeEntry } from '@/api/types'
import { useBreadcrumbs, type Crumb } from '@/composables/useBreadcrumbs'
import { ApiError } from '@/api/client'

const props = defineProps<{
  owner: string
  repoPath: string
}>()

const { setBreadcrumbs, clearBreadcrumbs } = useBreadcrumbs()

const repo = ref<Repository | null>(null)
const branches = ref<RefInfo[]>([])
const tree = ref<TreeEntry[]>([])
const readme = ref<BlobContent | null>(null)
const stats = ref<RepoStats | null>(null)
const error = ref('')
const loading = ref(true)
const copied = ref(false)

const refName = computed(() => repo.value?.default_branch || 'main')
const clone = computed(() => (repo.value ? cloneUrl(props.owner, props.repoPath) : ''))

const readmeHtml = computed(() => {
  if (!readme.value?.content || readme.value.is_binary) {
    return ''
  }
  return marked.parse(readme.value.content, { async: false }) as string
})

function setCrumbs() {
  const crumbs: Crumb[] = [{ label: props.owner, to: '/' }]
  props.repoPath.split('/').forEach((part, i, arr) => {
    const isLast = i === arr.length - 1
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
      : { label: part },
    )
  })
  setBreadcrumbs(crumbs)
}

async function load() {
  loading.value = true
  error.value = ''
  readme.value = null
  try {
    repo.value = await reposApi.get(props.owner, props.repoPath)
    setCrumbs()
    const ref = repo.value.default_branch || 'main'
    const [branchRes, treeRes, statsRes] = await Promise.all([
      reposApi.branches(props.owner, props.repoPath),
      reposApi.tree(props.owner, props.repoPath, { ref }),
      reposApi.stats(props.owner, props.repoPath, ref),
    ])
    branches.value = branchRes.branches ?? []
    tree.value = treeRes.tree ?? []
    stats.value = statsRes
    try {
      readme.value = await reposApi.readme(props.owner, props.repoPath, ref)
    } catch {
      readme.value = null
    }
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'Failed to load repository'
  } finally {
    loading.value = false
  }
}

async function copyClone() {
  try {
    await navigator.clipboard.writeText(clone.value)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 1500)
  } catch {}
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

watch(() => [props.owner, props.repoPath], load, { immediate: true })
onUnmounted(clearBreadcrumbs)
</script>

<template>
  <div>
    <p v-if="loading" class="text-sm text-ink-muted">Loading...</p>
    <p v-else-if="error" class="text-sm text-warn">{{ error }}</p>
    <template v-else-if="repo">
      <div class="mb-6 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 class="font-display text-3xl font-bold">{{ repo.name }}</h1>
          <p v-if="repo.description" class="mt-1 text-ink-muted">{{ repo.description }}</p>
          <div class="mt-3 flex flex-wrap gap-2 text-xs">
            <span class="rounded border border-line px-2 py-1 font-mono">{{ repo.default_branch }}</span>
            <span
              class="rounded border px-2 py-1 font-semibold uppercase tracking-wide"
              :class="repo.is_private ? 'border-warn/30 bg-warn-soft text-warn' : 'border-line text-ink-muted'"
            >
              {{ repo.is_private ? 'private' : 'public' }}
            </span>
            <span v-if="stats" class="rounded border border-line px-2 py-1 text-ink-muted">
              {{ stats.commit_count }} commits
            </span>
          </div>
        </div>
        <div class="flex flex-wrap gap-2">
          <RouterLink
            class="btn-ghost"
            :to="{
              name: 'repo-commits',
              params: { owner, repoPath }
            }"
          >
            Commits
          </RouterLink>
          <RouterLink
            class="btn-ghost"
            :to="{
              name: 'repo-tree',
              params: { owner, repoPath }, 
              query: { ref: refName }
            }"
          >
            Files
          </RouterLink>
          <RouterLink
            class="btn-ghost"
            :to="{
              name: 'repo-settings',
              params: { owner, repoPath }
            }"
          >
            Settings
          </RouterLink>
        </div>
      </div>

      <div class="panel mb-6 flex flex-wrap items-center gap-2 p-3">
        <span class="text-xs font-semibold uppercase tracking-wide text-ink-muted">Clone</span>
        <code class="min-w-0 flex-1 truncate rounded-md bg-paper px-2 py-1.5 font-mono text-sm">{{ clone }}</code>
        <button type="button" class="btn-ghost" @click="copyClone">{{ copied ? 'Copied' : 'Copy' }}</button>
      </div>

      <div class="mb-6 grid gap-4 lg:grid-cols-[1fr_220px]">
        <div>
          <h2 class="mb-2 text-xs font-semibold uppercase tracking-wide text-ink-muted">Files</h2>
          <ul v-if="tree.length" class="divide-y divide-line overflow-hidden rounded-lg border border-line bg-white/80">
            <li v-for="entry in tree" :key="entry.sha + entry.path">
              <RouterLink :to="entryLink(entry)" class="flex items-center gap-3 px-4 py-2.5 hover:bg-moss-soft/40">
                <span class="w-10 font-mono text-xs text-ink-muted">{{ entry.type }}</span>
                <span class="font-medium">{{ entry.name }}</span>
              </RouterLink>
            </li>
          </ul>
          <p v-else class="rounded-lg border border-dashed border-line px-4 py-8 text-center text-sm text-ink-muted">
            Empty repository - push a commit to get started.
          </p>
        </div>

        <aside class="space-y-4">
          <div class="panel p-3">
            <h2 class="mb-2 text-xs font-semibold uppercase tracking-wide text-ink-muted">Branches</h2>
            <ul v-if="branches.length" class="space-y-1 text-sm">
              <li v-for="b in branches" :key="b.name" class="font-mono">{{ b.name }}</li>
            </ul>
            <p v-else class="text-sm text-ink-muted">None yet</p>
          </div>
          <div v-if="stats && Object.keys(stats.languages).length" class="panel p-3">
            <h2 class="mb-2 text-xs font-semibold uppercase tracking-wide text-ink-muted">Languages</h2>
            <ul class="space-y-1 text-sm">
              <li v-for="(bytes, lang) in stats.languages" :key="lang" class="flex justify-between gap-2">
                <span>{{ lang }}</span>
                <span class="font-mono text-ink-muted">{{ bytes }}</span>
              </li>
            </ul>
          </div>
        </aside>
      </div>

      <section v-if="readme">
        <h2 class="mb-2 text-xs font-semibold uppercase tracking-wide text-ink-muted">{{ readme.path }}</h2>
        <article
          class="panel prose-repo overflow-x-auto p-5 text-sm leading-relaxed"
          v-html="readmeHtml"
        />
      </section>
    </template>
  </div>
</template>

<style scoped>
.prose-repo :deep(h1),
.prose-repo :deep(h2),
.prose-repo :deep(h3) {
  font-family: var(--font-display);
  font-weight: 700;
  margin: 1em 0 0.4em;
}

.prose-repo :deep(p) {
  margin: 0.6em 0;
}

.prose-repo :deep(pre) {
  background: var(--color-paper);
  border-radius: 0.375rem;
  padding: 0.75rem;
  overflow-x: auto;
  font-family: var(--font-mono);
  font-size: 0.85em;
}

.prose-repo :deep(code) {
  font-family: var(--font-mono);
  font-size: 0.9em;
}

.prose-repo :deep(a) {
  color: var(--color-moss);
  text-decoration: underline;
}
</style>
