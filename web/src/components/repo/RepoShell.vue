<script setup lang="ts">
import { computed } from 'vue'
import type { Repository } from '@/api/types'
import { useCanWrite } from '@/composables/useCanWrite'
import { useRepoSeo } from '@/composables/useRouteSeo'
import { normalizeRepoPath } from '@/lib/repoPath'
import Icon from '@/components/ui/Icon.vue'
import PageState from '@/components/ui/PageState.vue'
import RepoMirrorBadge from '@/components/repo/RepoMirrorBadge.vue'
import RepoTabBar from '@/components/repo/RepoTabBar.vue'
import type { PageError } from '@/i18n'

const props = defineProps<{
  repoPath: string
  tab: 'summary' | 'log' | 'commit' | 'refs' | 'issues' | 'pulls' | 'settings'
  repo: Repository | null
  loading?: boolean
  error?: PageError | string
}>()

const { canAdmin } = useCanWrite(() => props.repo)
useRepoSeo(() => props.repoPath)

function isNotFound(error: PageError | string | null | undefined) {
  return typeof error === 'object' && error?.kind === 'not_found'
}

const repoName = computed(() => props.repo?.name || normalizeRepoPath(props.repoPath) || props.repoPath)
const issuesEnabled = computed(() => props.repo?.issues_enabled !== false)
</script>

<template>
  <PageState
    v-if="loading || isNotFound(error) || (error && !repo)"
    :loading="!!loading"
    :error="error || ''"
  />
  <div v-else class="relative z-10 space-y-4">
    <div class="surface overflow-visible">
      <div class="relative z-20 flex min-w-0 flex-wrap items-center gap-x-1.5 gap-y-0.5 px-3 py-2 text-[90%] text-ink sm:px-4">
        <h1 class="min-w-0 truncate text-[1.1rem] font-semibold leading-snug sm:text-[1.15rem]">
          {{ repoName }}
        </h1>
        <span
          v-if="repo?.is_private"
          class="ml-1.5 inline-flex align-middle text-ink-muted"
          :title="$t('home.private')"
          :aria-label="$t('settings.private')"
        >
          <Icon name="lock" size="sm" />
        </span>
        <RepoMirrorBadge :repo-path="repoPath" :repo="repo" />
      </div>

      <RepoTabBar
        :repo-path="repoPath"
        :tab="tab"
        :issues-enabled="issuesEnabled"
        :can-admin="canAdmin"
      />
    </div>

    <div class="page-enter min-w-0">
      <slot v-if="repo" />
    </div>
  </div>
</template>
