<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import type { Repository } from '@/api/types'
import { useCanWrite } from '@/composables/useCanWrite'
import { folderLocation } from '@/lib/folderNav'
import { repoLocation } from '@/lib/repoNav'
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

function isNotFound(error: PageError | string | null | undefined) {
  return typeof error === 'object' && error?.kind === 'not_found'
}

const parts = computed(() => normalizeRepoPath(props.repoPath).split('/').filter(Boolean))
const folderSlug = computed(() => {
  if (props.repo?.folder_path) {
    return normalizeRepoPath(props.repo.folder_path)
  }

  return parts.value.length > 1 ? parts.value.slice(0, -1).join('/') : ''
})
const repoName = computed(() => parts.value[parts.value.length - 1] || props.repoPath)
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
        <RouterLink to="/" class="text-ink-muted no-underline hover:underline">{{ $t('nav.root') }}</RouterLink>
        <template v-if="folderSlug">
          <span class="text-ink-muted">/</span>
          <RouterLink
            :to="folderLocation(folderSlug)"
            class="text-ink-muted no-underline hover:underline"
          >
            {{ folderSlug }}
          </RouterLink>
        </template>
        <span class="text-ink-muted">/</span>
        <RouterLink class="text-[1.1rem] font-semibold leading-snug no-underline hover:underline sm:text-[1.15rem]" :to="repoLocation('repo', repoPath)">
          {{ repoName }}
        </RouterLink>
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
