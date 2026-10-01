<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { cloneUrl, reposApi, sshCloneUrl } from '@/api'
import type { BlobContent, RefInfo, Repository, RepoStats, TreeEntry } from '@/api/types'
import CloneMenu from '@/components/repo/CloneMenu.vue'
import FileTable from '@/components/repo/FileTable.vue'
import LanguageBar from '@/components/repo/LanguageBar.vue'
import ReadmePanel from '@/components/repo/ReadmePanel.vue'
import RepoShell from '@/components/repo/RepoShell.vue'
import PageState from '@/components/ui/PageState.vue'
import { localizeError } from '@/i18n'

const props = defineProps<{
  owner: string
  repoPath: string
}>()

const repo = ref<Repository | null>(null)
const branches = ref<RefInfo[]>([])
const tree = ref<TreeEntry[]>([])
const readme = ref<BlobContent | null>(null)
const stats = ref<RepoStats | null>(null)
const error = ref('')
const loading = ref(true)

const refName = computed(() => repo.value?.default_branch || 'main')
const httpsClone = computed(() => (repo.value ? cloneUrl(props.owner, props.repoPath) : ''))
const sshClone = computed(() => (repo.value ? sshCloneUrl(props.owner, props.repoPath) : ''))
const hasReadme = computed(() => !!readme.value?.content && !readme.value.is_binary)

async function load() {
  loading.value = true
  error.value = ''
  readme.value = null
  try {
    repo.value = await reposApi.get(props.owner, props.repoPath)
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
    error.value = localizeError(err, 'errors.loadRepository')
  } finally {
    loading.value = false
  }
}

watch(() => [props.owner, props.repoPath], load, { immediate: true })
</script>

<template>
  <RepoShell
    :owner="owner"
    :repo-path="repoPath"
    tab="code"
  >
    <PageState :loading="loading" :error="error">
      <div v-if="repo" class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_280px]">
        <div class="min-w-0 space-y-4">
          <FileTable
            :entries="tree"
            :owner="owner"
            :repo-path="repoPath"
            :ref-name="refName"
          >
            <template #meta>
              <RouterLink
                v-if="stats"
                :to="{
                  name: 'repo-commits',
                  params: { owner, repoPath }
                }"
                class="text-sm text-ink hover:text-accent hover:underline"
              >
                {{ $t('repo.commits', { n: stats.commit_count }) }}
              </RouterLink>
            </template>
            <template #actions>
              <CloneMenu v-if="httpsClone" :https-url="httpsClone" :ssh-url="sshClone" />
            </template>
          </FileTable>
          <ReadmePanel v-if="readme && hasReadme" :readme="readme" />
        </div>

        <aside class="space-y-4">
          <section class="rounded-md border border-line bg-white p-4">
            <h2 class="mb-2 text-sm font-semibold">{{ $t('repo.branches') }}</h2>
            <ul v-if="branches.length" class="space-y-1 font-mono text-xs">
              <li v-for="branch in branches" :key="branch.name">{{ branch.name }}</li>
            </ul>
            <p v-else class="text-sm text-ink-muted">{{ $t('repo.noneYet') }}</p>
          </section>
          <section v-if="stats && Object.keys(stats.languages).length" class="rounded-md border border-line bg-white p-4">
            <h2 class="mb-3 text-sm font-semibold">{{ $t('repo.languages') }}</h2>
            <LanguageBar :languages="stats.languages" />
          </section>
        </aside>
      </div>
    </PageState>
  </RepoShell>
</template>
