<script setup lang="ts">
import { ref, watch } from 'vue'
import { reposApi } from '@/api'
import type { CommitInfo, Repository } from '@/api/types'
import CommitList from '@/components/repo/CommitList.vue'
import RepoShell from '@/components/repo/RepoShell.vue'
import PageState from '@/components/ui/PageState.vue'
import { localizeError } from '@/i18n'

const props = defineProps<{
  owner: string
  repoPath: string
}>()

const repo = ref<Repository | null>(null)
const commits = ref<CommitInfo[]>([])
const error = ref('')
const loading = ref(true)
const offset = ref(0)
const limit = 30

async function load() {
  loading.value = true
  error.value = ''
  try {
    repo.value = await reposApi.get(props.owner, props.repoPath)
    const res = await reposApi.commits(props.owner, props.repoPath, {
      ref: repo.value.default_branch,
      offset: offset.value,
      limit,
    })
    commits.value = res.commits ?? []
  } catch (err) {
    error.value = localizeError(err, 'errors.loadCommits')
  } finally {
    loading.value = false
  }
}

watch(() => [props.owner, props.repoPath], () => {
  offset.value = 0
  load()
}, { immediate: true })
</script>

<template>
  <RepoShell :owner="owner" :repo-path="repoPath" tab="code">
    <div class="mb-3 flex items-center justify-between">
      <h1 class="text-xl font-semibold">{{ $t('commits.title') }}</h1>
    </div>
    <PageState :loading="loading" :error="error">
      <CommitList
        :commits="commits"
        :owner="owner"
        :repo-path="repoPath"
      />
      <div
        v-if="commits.length === limit || offset > 0"
        class="mt-4 flex gap-2"
      >
        <button
          type="button"
          class="btn-ghost"
          :disabled="offset === 0"
          @click="offset = Math.max(0, offset - limit); load()"
        >
          {{ $t('commits.newer') }}
        </button>
        <button
          type="button" class="btn-ghost"
          :disabled="commits.length < limit"
          @click="offset += limit; load()"
        >
          {{ $t('commits.older') }}
        </button>
      </div>
    </PageState>
  </RepoShell>
</template>
