<script setup lang="ts">
import { ref, watch } from 'vue'
import { reposApi } from '@/api'
import type { CommitInfo } from '@/api/types'
import CommitList from '@/components/repo/CommitList.vue'
import RepoShell from '@/components/repo/RepoShell.vue'
import PageState from '@/components/ui/PageState.vue'
import { useRepo } from '@/composables/useRepo'
import type { PageError } from '@/i18n'
import { pageError } from '@/i18n'

const props = defineProps<{
  repoPath: string
}>()

const { repo, loading: repoLoading, error: repoError } = useRepo(() => props.repoPath)

const commits = ref<CommitInfo[]>([])
const error = ref<PageError | string>('')
const dataLoading = ref(false)
const offset = ref(0)
const limit = 30

async function loadCommits() {
  dataLoading.value = true
  error.value = ''
  try {
    const res = await reposApi.commits(props.repoPath, {
      offset: offset.value,
      limit,
    })
    commits.value = res.commits ?? []
  } catch (err) {
    error.value = pageError(err, 'errors.loadCommits')
  } finally {
    dataLoading.value = false
  }
}

watch(
  () => props.repoPath,
  () => {
    offset.value = 0
    loadCommits()
  },
  { immediate: true },
)
</script>

<template>
  <RepoShell 
    :repo-path="repoPath" 
    tab="log" 
    :repo="repo" 
    :loading="repoLoading" 
    :error="repoError"
  >
    <PageState :loading="dataLoading" :error="error">
      <CommitList :commits="commits" :repo-path="repoPath" />
      <div v-if="commits.length === limit || offset > 0" class="mt-4 flex gap-2">
        <button
          type="button"
          class="inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-ink hover:bg-row-hover disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="offset === 0"
          @click="offset = Math.max(0, offset - limit); loadCommits()"
        >
          {{ $t('commits.newer') }}
        </button>
        <button
          type="button"
          class="inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-ink hover:bg-row-hover disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="commits.length < limit"
          @click="offset += limit; loadCommits()"
        >
          {{ $t('commits.older') }}
        </button>
      </div>
    </PageState>
  </RepoShell>
</template>
