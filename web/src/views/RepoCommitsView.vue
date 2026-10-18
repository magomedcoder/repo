<script setup lang="ts">
import { useRepoOwner } from '@/composables/useRepoOwner'
import { btnGhost } from '@/lib/classes'
import { ref, watch } from 'vue'
import { reposApi } from '@/api'
import type { CommitInfo, Repository } from '@/api/types'
import CommitList from '@/components/repo/CommitList.vue'
import RepoShell from '@/components/repo/RepoShell.vue'
import PageState from '@/components/ui/PageState.vue'
import { pageError } from '@/i18n'

const owner = useRepoOwner()

const props = defineProps<{
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
    repo.value = await reposApi.get(owner.value, props.repoPath)
    const res = await reposApi.commits(owner.value, props.repoPath, {
      ref: repo.value.default_branch,
      offset: offset.value,
      limit,
    })
    commits.value = res.commits ?? []
  } catch (err) {
    error.value = pageError(err, 'errors.loadCommits')
  } finally {
    loading.value = false
  }
}

watch(() => [owner.value, props.repoPath], () => {
  offset.value = 0
  load()
}, { immediate: true })
</script>

<template>
  <RepoShell :owner="owner" :repo-path="repoPath" tab="code">
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
          :class="btnGhost"
          :disabled="offset === 0"
          @click="offset = Math.max(0, offset - limit); load()"
        >
          {{ $t('commits.newer') }}
        </button>
        <button
          type="button" :class="btnGhost"
          :disabled="commits.length < limit"
          @click="offset += limit; load()"
        >
          {{ $t('commits.older') }}
        </button>
      </div>
    </PageState>
  </RepoShell>
</template>
