<script setup lang="ts">
import { dataTable } from '@/lib/classes'
import { computed, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { cloneUrl, reposApi } from '@/api'
import type { BlobContent, CommitInfo, RefInfo, RepoStats, TreeEntry } from '@/api/types'
import CommitList from '@/components/repo/CommitList.vue'
import FileTable from '@/components/repo/FileTable.vue'
import LanguageBar from '@/components/repo/LanguageBar.vue'
import ReadmePanel from '@/components/repo/ReadmePanel.vue'
import PageState from '@/components/ui/PageState.vue'
import { repoPathMatches, useInjectedRepo } from '@/composables/useInjectedRepo'
import type { PageError } from '@/i18n'
import { pageError } from '@/i18n'

const props = defineProps<{
  repoPath: string
}>()

const repo = useInjectedRepo()
const branches = ref<RefInfo[]>([])
const commits = ref<CommitInfo[]>([])
const tree = ref<TreeEntry[]>([])
const readme = ref<BlobContent | null>(null)
const stats = ref<RepoStats | null>(null)
const error = ref<PageError | string>('')
const dataLoading = ref(false)

const loading = computed(
  () => !repo.value || !repoPathMatches(repo.value, props.repoPath) || dataLoading.value,
)

const refName = computed(() => repo.value?.default_branch || 'main')
const httpsClone = computed(() => (repo.value ? cloneUrl(props.repoPath) : ''))
const hasReadme = computed(() => !!readme.value?.content && !readme.value.is_binary)

async function loadParallel(ref: string) {
  const [branchRes, treeRes, statsRes, commitRes] = await Promise.all([
    reposApi.branches(props.repoPath),
    reposApi.tree(props.repoPath, { ref }),
    reposApi.stats(props.repoPath, ref),
    reposApi.commits(props.repoPath, { ref, limit: 8 }).catch(() => ({ commits: [] as CommitInfo[] })),
  ])
  branches.value = branchRes.branches ?? []
  commits.value = commitRes.commits ?? []
  tree.value = treeRes.tree ?? []
  stats.value = statsRes
  try {
    readme.value = await reposApi.readme(props.repoPath, ref)
  } catch {
    readme.value = null
  }
}

async function loadData() {
  if (!repo.value || !repoPathMatches(repo.value, props.repoPath)) {
    return
  }
  dataLoading.value = true
  error.value = ''
  readme.value = null
  try {
    const ref = repo.value.default_branch || 'main'
    const summary = await reposApi.summary(props.repoPath, ref)
    if (summary) {
      branches.value = summary.branches ?? []
      commits.value = summary.recent_commits ?? []
      tree.value = summary.tree ?? []
      stats.value = summary.stats ?? null
      readme.value = summary.readme ?? null
      return
    }
    await loadParallel(ref)
  } catch (err) {
    error.value = pageError(err, 'errors.loadRepository')
  } finally {
    dataLoading.value = false
  }
}

watch([() => props.repoPath, repo], loadData, { immediate: true })
</script>

<template>
  <PageState :loading="loading" :error="error">
    <div v-if="repo && repoPathMatches(repo, repoPath)">
      <table :class="[dataTable, 'mb-3']">
        <tbody>
          <tr>
            <th class="w-36">{{ $t('repo.clone') }}</th>
            <td class="font-mono">{{ httpsClone }}</td>
          </tr>
          <tr>
            <th>{{ $t('index.owner') }}</th>
            <td>{{ repo.owner }}</td>
          </tr>
          <tr>
            <th>{{ $t('index.description') }}</th>
            <td>{{ repo.description }}</td>
          </tr>
          <tr>
            <th>{{ $t('repo.branches') }}</th>
            <td>
              <template v-if="branches.length">
                <span v-for="(branch, i) in branches" :key="branch.name">
                  <span v-if="i">, </span>{{ branch.name }}
                </span>
              </template>
              <span v-else>{{ $t('repo.noneYet') }}</span>
            </td>
          </tr>
          <tr v-if="stats">
            <th>{{ $t('index.log') }}</th>
            <td>
              <RouterLink :to="{ name: 'repo-commits', params: { repoPath } }">
                {{ $t('repo.commits', { n: stats.commit_count }) }}
              </RouterLink>
            </td>
          </tr>
        </tbody>
      </table>
      <CommitList class="mb-3" :commits="commits" :repo-path="repoPath" />
      <FileTable :entries="tree" :repo-path="repoPath" :ref-name="refName" />
      <div v-if="stats && Object.keys(stats.languages).length" class="mt-3 px-2">
        <p class="mb-1">{{ $t('repo.languages') }}</p>
        <LanguageBar :languages="stats.languages" />
      </div>
      <ReadmePanel v-if="readme && hasReadme" :readme="readme" />
    </div>
  </PageState>
</template>
