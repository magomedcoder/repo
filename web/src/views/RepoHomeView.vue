<script setup lang="ts">
import { useRepoOwner } from '@/composables/useRepoOwner'
import { dataTable } from '@/lib/classes'
import { computed, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { cloneUrl, reposApi } from '@/api'
import type { BlobContent, CommitInfo, RefInfo, Repository, RepoStats, TreeEntry } from '@/api/types'
import CommitList from '@/components/repo/CommitList.vue'
import FileTable from '@/components/repo/FileTable.vue'
import LanguageBar from '@/components/repo/LanguageBar.vue'
import ReadmePanel from '@/components/repo/ReadmePanel.vue'
import RepoShell from '@/components/repo/RepoShell.vue'
import PageState from '@/components/ui/PageState.vue'
import { pageError } from '@/i18n'

const owner = useRepoOwner()

const props = defineProps<{
  repoPath: string
}>()

const repo = ref<Repository | null>(null)
const branches = ref<RefInfo[]>([])
const commits = ref<CommitInfo[]>([])
const tree = ref<TreeEntry[]>([])
const readme = ref<BlobContent | null>(null)
const stats = ref<RepoStats | null>(null)
const error = ref('')
const loading = ref(true)

const refName = computed(() => repo.value?.default_branch || 'main')
const httpsClone = computed(() => (repo.value ? cloneUrl(props.repoPath) : ''))
const hasReadme = computed(() => !!readme.value?.content && !readme.value.is_binary)

async function load() {
  loading.value = true
  error.value = ''
  readme.value = null
  try {
    repo.value = await reposApi.get(owner.value, props.repoPath)
    const ref = repo.value.default_branch || 'main'
    const [branchRes, treeRes, statsRes, commitRes] = await Promise.all([
      reposApi.branches(owner.value, props.repoPath),
      reposApi.tree(owner.value, props.repoPath, { ref }),
      reposApi.stats(owner.value, props.repoPath, ref),
      reposApi.commits(owner.value, props.repoPath, { ref, limit: 8 }).catch(() => ({ commits: [] as CommitInfo[] })),
    ])
    branches.value = branchRes.branches ?? []
    commits.value = commitRes.commits ?? []
    tree.value = treeRes.tree ?? []
    stats.value = statsRes
    try {
      readme.value = await reposApi.readme(owner.value, props.repoPath, ref)
    } catch {
      readme.value = null
    }
  } catch (err) {
    error.value = pageError(err, 'errors.loadRepository')
  } finally {
    loading.value = false
  }
}

watch(() => [owner.value, props.repoPath], load, { immediate: true })
</script>

<template>
  <RepoShell
    :owner="owner"
    :repo-path="repoPath"
    tab="code"
  >
    <PageState :loading="loading" :error="error">
      <div v-if="repo">
        <table :class="[dataTable, 'mb-3']">
          <tbody>
            <tr>
              <th class="w-36">{{ $t('repo.clone') }}</th>
              <td class="font-mono">{{ httpsClone }}</td>
            </tr>
            <tr>
              <th>{{ $t('index.owner') }}</th>
              <td>{{ owner }}</td>
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
        <CommitList
          class="mb-3"
          :commits="commits"
          :owner="owner"
          :repo-path="repoPath"
        />
        <FileTable
          :entries="tree"
          :owner="owner"
          :repo-path="repoPath"
          :ref-name="refName"
        />
        <div v-if="stats && Object.keys(stats.languages).length" class="mt-3 px-2">
          <p class="mb-1">{{ $t('repo.languages') }}</p>
          <LanguageBar :languages="stats.languages" />
        </div>
        <ReadmePanel v-if="readme && hasReadme" :readme="readme" />
      </div>
    </PageState>
  </RepoShell>
</template>
