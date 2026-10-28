<script setup lang="ts">
import { ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { reposApi } from '@/api'
import type { CommitDiff, CommitInfo } from '@/api/types'
import DiffFileList from '@/components/repo/DiffFileList.vue'
import RepoShell from '@/components/repo/RepoShell.vue'
import PageState from '@/components/ui/PageState.vue'
import { useRepo } from '@/composables/useRepo'
import type { PageError } from '@/i18n'
import { formatDate, pageError } from '@/i18n'
import { repoLocation } from '@/lib/repoNav'

const props = defineProps<{
  repoPath: string
  sha: string
}>()

const { repo, loading: repoLoading, error: repoError } = useRepo(() => props.repoPath)

const commit = ref<CommitInfo | null>(null)
const diff = ref<CommitDiff | null>(null)
const error = ref<PageError | string>('')
const loading = ref(true)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [info, patch] = await Promise.all([
      reposApi.commit(props.repoPath, props.sha),
      reposApi.diff(props.repoPath, props.sha),
    ])
    commit.value = info
    diff.value = patch
  } catch (err) {
    error.value = pageError(err, 'errors.loadCommit')
  } finally {
    loading.value = false
  }
}

watch(() => [props.repoPath, props.sha], load, { immediate: true })
</script>

<template>
  <RepoShell 
    :repo-path="repoPath" 
    tab="commit" 
    :repo="repo" 
    :loading="repoLoading" 
    :error="repoError"
  >
    <PageState :loading="loading" :error="error">
      <template v-if="commit">
        <article class="overflow-hidden border border-line bg-paper mb-4">
          <header class="flex items-center gap-2 border-b border-line bg-path px-3 py-1.5 text-[10pt] text-ink">{{ $t('commit.title') }}</header>
          <div class="border-b border-line px-4 py-3">
            <h1 class="text-base font-semibold text-ink">{{ commit.message.split('\n')[0] }}</h1>
            <p class="mt-1 text-xs text-ink-muted">
              {{ commit.author_name }}
              &lt;{{ commit.author_email }}&gt;
              {{ formatDate(commit.authored_at) }}
            </p>
          </div>
          <pre v-if="commit.message.includes('\n')" class="whitespace-pre-wrap px-4 py-3 text-sm">{{ commit.message }}</pre>
          <div class="flex items-center justify-between border-t border-line px-4 py-2 text-xs">
            <span class="font-mono text-ink-muted">{{ commit.sha }}</span>
            <RouterLink
              :to="repoLocation('repo-commits', repoPath)"
              class="text-ink hover:underline"
            >
              {{ $t('commit.all') }}
            </RouterLink>
          </div>
        </article>
        <h2 v-if="diff" class="mb-2 text-sm font-semibold">{{ $t('commit.filesChanged', { n: diff.files.length }) }}</h2>
        <div v-if="diff" class="space-y-3">
          <DiffFileList :files="diff.files" />
        </div>
      </template>
    </PageState>
  </RepoShell>
</template>
