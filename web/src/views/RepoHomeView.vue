<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { cloneUrl, reposApi } from '@/api'
import { ApiError } from '@/api/client'
import type { BlobContent, RefInfo, RepoStats, TreeEntry } from '@/api/types'
import FileTable from '@/components/repo/FileTable.vue'
import LanguageBar from '@/components/repo/LanguageBar.vue'
import ReadmePanel from '@/components/repo/ReadmePanel.vue'
import RepoShell from '@/components/repo/RepoShell.vue'
import PageState from '@/components/ui/PageState.vue'
import { useRepo } from '@/composables/useRepo'
import type { PageError } from '@/i18n'
import { pageError } from '@/i18n'
import { repoLocation } from '@/lib/repoNav'

const props = defineProps<{
  repoPath: string
}>()

const route = useRoute()
const { repo, loading, error: repoError } = useRepo(() => props.repoPath)
const branches = ref<RefInfo[]>([])
const tree = ref<TreeEntry[]>([])
const readme = ref<BlobContent | null>(null)
const stats = ref<RepoStats | null>(null)
const error = ref<PageError | string>('')
const dataLoading = ref(false)

const refName = computed(() => (route.query.ref as string) || repo.value?.default_branch || 'main')
const dirPath = computed(() => (route.query.path as string) || '')
const browsingSubdir = computed(() => !!dirPath.value)

const parentTo = computed(() => {
  if (!dirPath.value) {
    return null
  }

  const parts = dirPath.value.split('/').filter(Boolean)
  parts.pop()
  const path = parts.join('/')
  return repoLocation('repo', props.repoPath, {
    query: {
      ref: refName.value,
      path: path || undefined,
    },
  })
})

const hasReadme = () => !!readme.value?.content && !readme.value.is_binary

function summaryAllowsLegacyFallback(err: unknown): boolean {
  if (!(err instanceof ApiError)) {
    return false
  }

  if (err.status === 404) {
    return true
  }

  const code = err.code
  if (code === 'not_found' || code === 'unimplemented') {
    return true
  }

  return !!code && code.endsWith('_not_found')
}

async function loadRoot(ref: string) {
  try {
    const summary = await reposApi.summary(props.repoPath, ref)
    branches.value = summary.branches ?? []
    tree.value = summary.tree ?? []
    stats.value = summary.stats ?? null
    readme.value = summary.readme ?? null
  } catch (err) {
    if (!summaryAllowsLegacyFallback(err)) {
      throw err
    }

    const [branchRes, treeRes, statsRes] = await Promise.all([
      reposApi.branches(props.repoPath),
      reposApi.tree(props.repoPath, { ref }),
      reposApi.stats(props.repoPath, ref),
    ])
    branches.value = branchRes.branches ?? []
    tree.value = treeRes.tree ?? []
    stats.value = statsRes
    try {
      readme.value = await reposApi.readme(props.repoPath, ref)
    } catch {
      readme.value = null
    }
  }
}

async function loadDir(ref: string, path: string) {
  const res = await reposApi.tree(props.repoPath, { ref, path })
  tree.value = res.tree ?? []
  readme.value = null
}

async function loadData() {
  if (!repo.value) {
    return
  }

  dataLoading.value = true
  error.value = ''
  try {
    const ref = refName.value
    if (dirPath.value) {
      await loadDir(ref, dirPath.value)
    } else {
      await loadRoot(ref)
    }
  } catch (err) {
    error.value = pageError(err, dirPath.value ? 'errors.loadTree' : 'errors.loadRepository')
    tree.value = []
  } finally {
    dataLoading.value = false
  }
}

watch(
  () => [props.repoPath, repo.value, route.query.ref, route.query.path] as const,
  loadData,
  { immediate: true },
)
</script>

<template>
  <RepoShell
    :repo-path="repoPath"
    tab="summary"
    :repo="repo"
    :loading="loading"
    :error="repoError"
  >
    <PageState :loading="dataLoading" :error="error">
      <template v-if="!browsingSubdir">
        <div class="mb-3 space-y-0.5 font-mono text-[90%] leading-snug">
          <div v-if="repo?.description" class="flex min-w-0 flex-wrap items-baseline gap-x-2">
            <span class="shrink-0 text-ink-muted">{{ $t('index.description') }}</span>
            <span class="min-w-0 font-sans text-ink">{{ repo.description }}</span>
          </div>
          <div class="flex min-w-0 flex-wrap items-baseline gap-x-2">
            <span class="shrink-0 text-ink-muted">{{ $t('repo.clone') }}</span>
            <span class="min-w-0 break-all text-ink">{{ cloneUrl(repoPath) }}</span>
          </div>
          <div class="flex min-w-0 flex-wrap items-baseline gap-x-2">
            <span class="shrink-0 text-ink-muted">{{ $t('repo.branches') }}</span>
            <span class="min-w-0 text-ink">
              <template v-if="branches.length">
                <span v-for="(branch, i) in branches" :key="branch.name">
                  <span v-if="i">, </span>{{ branch.name }}
                </span>
              </template>
              <span v-else class="text-ink-muted">{{ $t('repo.noneYet') }}</span>
            </span>
          </div>
          <div v-if="stats" class="flex min-w-0 flex-wrap items-baseline gap-x-2">
            <span class="shrink-0 text-ink-muted">{{ $t('index.log') }}</span>
            <RouterLink
              :to="repoLocation('repo-commits', repoPath)"
              class="text-ink no-underline hover:underline"
            >
              {{ $t('repo.commits', { n: stats.commit_count }) }}
            </RouterLink>
          </div>
          <div
            v-if="stats && Object.keys(stats.languages).length"
            class="flex min-w-0 flex-wrap items-baseline gap-x-2"
          >
            <span class="shrink-0 text-ink-muted">{{ $t('repo.languages') }}</span>
            <LanguageBar :languages="stats.languages" />
          </div>
        </div>
      </template>

      <section class="mb-4">
        <FileTable
          :entries="tree"
          :repo-path="repoPath"
          :ref-name="refName"
          :path="dirPath || undefined"
          :parent-to="parentTo"
        />
      </section>

      <ReadmePanel
        v-if="!browsingSubdir && readme && hasReadme()"
        :readme="readme"
      />
    </PageState>
  </RepoShell>
</template>
