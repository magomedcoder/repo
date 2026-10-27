<script setup lang="ts">
import { btnGhost } from '@/lib/classes'
import { computed, ref, watch } from 'vue'
import { reposApi } from '@/api'
import type { CommitInfo } from '@/api/types'
import CommitList from '@/components/repo/CommitList.vue'
import PageState from '@/components/ui/PageState.vue'
import { repoPathMatches, useInjectedRepo } from '@/composables/useInjectedRepo'
import type { PageError } from '@/i18n'
import { pageError } from '@/i18n'

const props = defineProps<{
  repoPath: string
}>()

const repo = useInjectedRepo()
const commits = ref<CommitInfo[]>([])
const error = ref<PageError | string>('')
const dataLoading = ref(false)
const offset = ref(0)
const limit = 30

const loading = computed(
  () => !repo.value || !repoPathMatches(repo.value, props.repoPath) || dataLoading.value,
)

async function loadCommits() {
  if (!repo.value || !repoPathMatches(repo.value, props.repoPath)) {
    return
  }
  dataLoading.value = true
  error.value = ''
  try {
    const res = await reposApi.commits(props.repoPath, {
      ref: repo.value.default_branch,
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

watch([() => props.repoPath, repo], () => {
  offset.value = 0
  loadCommits()
}, { immediate: true })
</script>

<template>
  <PageState :loading="loading" :error="error">
    <CommitList :commits="commits" :repo-path="repoPath" />
    <div v-if="commits.length === limit || offset > 0" class="mt-4 flex gap-2">
      <button
        type="button"
        :class="btnGhost"
        :disabled="offset === 0"
        @click="offset = Math.max(0, offset - limit); loadCommits()"
      >
        {{ $t('commits.newer') }}
      </button>
      <button
        type="button"
        :class="btnGhost"
        :disabled="commits.length < limit"
        @click="offset += limit; loadCommits()"
      >
        {{ $t('commits.older') }}
      </button>
    </div>
  </PageState>
</template>
