<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { reposApi } from '@/api'
import type { BlobContent, CommitInfo, RepoStats, TreeEntry } from '@/api/types'
import CloneButton from '@/components/repo/CloneButton.vue'
import EmptyRepoGuide from '@/components/repo/EmptyRepoGuide.vue'
import FileTable from '@/components/repo/FileTable.vue'
import LanguageBar from '@/components/repo/LanguageBar.vue'
import ReadmePanel from '@/components/repo/ReadmePanel.vue'
import RepoShell from '@/components/repo/RepoShell.vue'
import Icon from '@/components/ui/Icon.vue'
import PageState from '@/components/ui/PageState.vue'
import { useRepo } from '@/composables/useRepo'
import type { PageError } from '@/i18n'
import { i18n, pageError } from '@/i18n'
import { idleAge } from '@/lib/idle'
import { repoLocation } from '@/lib/repoNav'

const props = defineProps<{
  repoPath: string
}>()

const route = useRoute()
const { repo, loading, error: repoError } = useRepo(() => props.repoPath)
const tree = ref<TreeEntry[]>([])
const readme = ref<BlobContent | null>(null)
const stats = ref<RepoStats | null>(null)
const latest = ref<CommitInfo | null>(null)
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

const isEmptyRepo = computed(() => !browsingSubdir.value && !dataLoading.value && !error.value && tree.value.length === 0 && !latest.value)

const canPush = computed(() => !!repo.value?.can_write)

const commitLabel = computed(() => {
  const n = stats.value?.commit_count ?? 0
  return i18n.global.t('repo.commits', n)
})

const latestMessage = computed(() => {
  if (!latest.value?.message) {
    return ''
  }
  return latest.value.message.split('\n')[0] || ''
})

const latestSha = computed(() => (latest.value?.sha || '').slice(0, 7))

async function loadRoot(ref: string) {
  const [summary, commitsRes] = await Promise.all([
    reposApi.summary(props.repoPath, ref),
    reposApi.commits(props.repoPath, { ref, limit: 1 }).catch(() => ({ commits: [] as CommitInfo[] })),
  ])
  tree.value = summary.tree ?? []
  stats.value = summary.stats ?? null
  readme.value = summary.readme ?? null
  latest.value = commitsRes.commits?.[0] ?? null
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
      <EmptyRepoGuide
        v-if="isEmptyRepo && canPush"
        :repo-path="repoPath"
        :default-branch="repo?.default_branch"
      />
      <div
        v-else
        class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_280px]"
      >
        <div class="min-w-0 space-y-4">
          <section class="min-w-0">
            <FileTable
              :entries="tree"
              :repo-path="repoPath"
              :ref-name="refName"
              :path="dirPath || undefined"
              :parent-to="parentTo"
            >
              <template #actions>
                <CloneButton :repo-path="repoPath" />
              </template>

              <template v-if="!browsingSubdir && latest" #banner>
                <div class="flex flex-wrap items-center justify-between gap-2 border-b border-line/80 bg-white/25 px-3 py-2 text-[12px]">
                  <div class="min-w-0 flex-1">
                    <span class="font-semibold text-ink">{{ latest.author_name }}</span>
                    <RouterLink
                      :to="repoLocation('repo-commit', repoPath, { sha: latest.sha })"
                      class="ml-1.5 text-ink no-underline hover:text-link hover:underline"
                    >
                      {{ latestMessage }}
                    </RouterLink>
                  </div>
                  <div class="flex shrink-0 flex-wrap items-center gap-3 text-ink-muted">
                    <RouterLink
                      :to="repoLocation('repo-commit', repoPath, { sha: latest.sha })"
                      class="font-mono text-ink-muted no-underline hover:text-link hover:underline"
                    >
                      {{ latestSha }}
                    </RouterLink>
                    <span v-if="latest.authored_at">{{ idleAge(latest.authored_at) }}</span>
                    <RouterLink
                      v-if="stats"
                      :to="repoLocation('repo-commits', repoPath)"
                      class="inline-flex items-center gap-1 text-ink-muted no-underline hover:text-link hover:underline"
                    >
                      <Icon name="commit" size="sm" />
                      {{ commitLabel }}
                    </RouterLink>
                  </div>
                </div>
              </template>
            </FileTable>
          </section>

          <ReadmePanel
            v-if="!browsingSubdir && readme && hasReadme()"
            :readme="readme"
          />
        </div>

        <aside
          v-if="!browsingSubdir"
          class="surface h-fit space-y-4 p-3 sm:p-4"
        >
          <div>
            <h2 class="mb-2 text-sm font-semibold text-ink">
              {{ $t('repo.about') }}
            </h2>
            <p v-if="repo?.description" class="text-sm leading-relaxed text-ink">
              {{ repo.description }}
            </p>
            <p v-else class="text-sm text-ink-muted">
              {{ $t('repo.noneYet') }}
            </p>
          </div>

          <div
            v-if="stats && Object.keys(stats.languages).length"
            class="border-t border-line/70 pt-3"
          >
            <h3 class="mb-2 text-sm font-semibold text-ink">
              {{ $t('repo.languages') }}
            </h3>
            <LanguageBar :languages="stats.languages" />
          </div>
        </aside>
      </div>
    </PageState>
  </RepoShell>
</template>
