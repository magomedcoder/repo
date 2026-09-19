<script setup lang="ts">
import { ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { reposApi } from '@/api'
import type { CommitDiff, CommitInfo } from '@/api/types'
import DiffFileList from '@/components/repo/DiffFileList.vue'
import RepoShell from '@/components/repo/RepoShell.vue'
import PageState from '@/components/ui/PageState.vue'
import { formatDate, localizeError } from '@/i18n'

const props = defineProps<{
  owner: string
  repoPath: string
  sha: string
}>()

const commit = ref<CommitInfo | null>(null)
const diff = ref<CommitDiff | null>(null)
const error = ref('')
const loading = ref(true)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [info, patch] = await Promise.all([
      reposApi.commit(props.owner, props.repoPath, props.sha),
      reposApi.diff(props.owner, props.repoPath, props.sha),
    ])
    commit.value = info
    diff.value = patch
  } catch (err) {
    error.value = localizeError(err, 'errors.loadCommit')
  } finally {
    loading.value = false
  }
}

watch(() => [props.owner, props.repoPath, props.sha], load, { immediate: true })
</script>

<template>
  <RepoShell
    :owner="owner"
    :repo-path="repoPath"
    tab="code"
  >
    <PageState :loading="loading" :error="error">
      <template v-if="commit">
        <article class="mb-4 overflow-hidden rounded-md border border-line bg-white">
          <header class="border-b border-line bg-paper px-4 py-3">
            <h1 class="text-xl font-semibold">{{ commit.message.split('\n')[0] }}</h1>
            <p class="mt-1 text-xs text-ink-muted">
              {{ commit.author_name }}
              &lt;{{ commit.author_email }}&gt;
              {{ formatDate(commit.authored_at) }}
            </p>
          </header>
          <pre v-if="commit.message.includes('\n')" class="whitespace-pre-wrap px-4 py-3 text-sm">{{ commit.message }}</pre>
          <div class="flex items-center justify-between border-t border-line px-4 py-2 text-xs">
            <span class="font-mono text-ink-muted">{{ commit.sha }}</span>
            <RouterLink
              :to="{
                name: 'repo-commits',
                params: { owner, repoPath }
              }" class="text-accent hover:underline"
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
