<script setup lang="ts">
import { ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
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

const { repo, loading, error: repoError } = useRepo(() => props.repoPath)
const branches = ref<RefInfo[]>([])
const tree = ref<TreeEntry[]>([])
const readme = ref<BlobContent | null>(null)
const stats = ref<RepoStats | null>(null)
const error = ref<PageError | string>('')
const dataLoading = ref(false)

const hasReadme = () => !!readme.value?.content && !readme.value.is_binary

async function loadParallel(refName: string) {
  const [branchRes, treeRes, statsRes] = await Promise.all([
    reposApi.branches(props.repoPath),
    reposApi.tree(props.repoPath, { ref: refName }),
    reposApi.stats(props.repoPath, refName),
  ])
  branches.value = branchRes.branches ?? []
  tree.value = treeRes.tree ?? []
  stats.value = statsRes
  try {
    readme.value = await reposApi.readme(props.repoPath, refName)
  } catch {
    readme.value = null
  }
}

async function loadData() {
  if (!repo.value) {
    return
  }

  dataLoading.value = true
  error.value = ''
  readme.value = null
  try {
    const refName = repo.value.default_branch || 'main'
    try {
      const summary = await reposApi.summary(props.repoPath, refName)
      branches.value = summary.branches ?? []
      tree.value = summary.tree ?? []
      stats.value = summary.stats ?? null
      readme.value = summary.readme ?? null
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        throw err
      }
      await loadParallel(refName)
    }
  } catch (err) {
    error.value = pageError(err, 'errors.loadRepository')
  } finally {
    dataLoading.value = false
  }
}

watch([() => props.repoPath, repo], loadData, { immediate: true })
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
      <div class="mb-4 space-y-1 font-mono text-[90%] leading-[1.55]">
        <div class="flex min-w-0 flex-wrap items-baseline gap-x-3 gap-y-0.5">
          <span class="shrink-0 text-ink-muted">{{ $t('repo.clone') }}</span>
          <span class="min-w-0 break-all text-ink">{{ cloneUrl(repoPath) }}</span>
        </div>
        <div class="flex min-w-0 flex-wrap items-baseline gap-x-3 gap-y-0.5">
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
        <div v-if="stats" class="flex min-w-0 flex-wrap items-baseline gap-x-3 gap-y-0.5">
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
          class="flex min-w-0 flex-wrap items-baseline gap-x-3 gap-y-0.5"
        >
          <span class="shrink-0 text-ink-muted">{{ $t('repo.languages') }}</span>
          <LanguageBar :languages="stats.languages" />
        </div>
      </div>

      <section class="mb-4">
        <h2 class="mb-1 text-[85%] font-medium uppercase tracking-wide text-ink-muted">
          {{ $t('index.tree') }}
        </h2>
        <FileTable
          tree
          :entries="tree"
          :repo-path="repoPath"
          :ref-name="repo?.default_branch || 'main'"
        />
      </section>

      <ReadmePanel v-if="readme && hasReadme()" :readme="readme" />
    </PageState>
  </RepoShell>
</template>
