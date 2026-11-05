<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import type { Repository } from '@/api/types'
import { useCanWrite } from '@/composables/useCanWrite'
import { folderLocation } from '@/lib/folderNav'
import { repoLocation } from '@/lib/repoNav'
import { normalizeRepoPath } from '@/lib/repoPath'
import PageState from '@/components/ui/PageState.vue'
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
const mirrorDirection = computed(() => props.repo?.mirror_direction || '')
const mirrorURL = computed(() => (props.repo?.mirror_url || '').trim())
const mirrorLinkLabel = computed(() => {
  const raw = mirrorURL.value
  if (!raw) {
    return ''
  }
  
  try {
    const u = new URL(raw)
    const path = u.pathname.replace(/\.git$/, '').replace(/^\/+|\/+$/g, '')
    return path ? `${u.host}/${path}` : u.host
  } catch {
    return raw.replace(/^https?:\/\//, '').replace(/\.git$/, '')
  }
})

function active(name: string) {
  return props.tab === name
}
</script>

<template>
  <PageState
    v-if="loading || isNotFound(error) || (error && !repo)"
    :loading="!!loading"
    :error="error || ''"
  />
  <div v-else>
    <div class="mb-0 font-mono text-[90%] text-ink">
      <RouterLink to="/" class="text-ink-muted no-underline hover:underline">{{ $t('nav.root') }}</RouterLink>
      <template v-if="folderSlug">
        <span> / </span>
        <RouterLink
          :to="folderLocation(folderSlug)"
          class="text-ink-muted no-underline hover:underline"
        >
          {{ folderSlug }}
        </RouterLink>
      </template>
      <span> / </span>
      <RouterLink class="font-bold no-underline hover:underline" :to="repoLocation('repo', repoPath)">
        {{ repoName }}
      </RouterLink>
    </div>

    <p v-if="mirrorDirection === 'pull' || mirrorDirection === 'push'" class="mt-1 text-[80%] text-ink-muted">
      <span>{{ mirrorDirection === 'pull' ? $t('repo.mirrorPullPrefix') : $t('repo.mirrorPushPrefix') }}</span>
      <a
        v-if="mirrorURL"
        :href="mirrorURL"
        class="text-ink-muted underline decoration-line underline-offset-2 hover:text-ink"
        target="_blank"
        rel="noopener noreferrer"
      >{{ mirrorLinkLabel }}</a>
      <span v-else>{{ $t('repo.mirrorRemoteUnknown') }}</span>
    </p>

    <nav class="m-0 mt-1.5 mb-0 flex w-full flex-wrap items-stretch gap-0 border-b border-line" :aria-label="$t('nav.repo')">
      <RouterLink :to="repoLocation('repo', repoPath)" :class="['relative -mb-px inline-block cursor-pointer border-b-2 border-transparent px-3 py-1.5 text-[100%] text-ink-muted no-underline hover:text-ink hover:no-underline', active('summary') && 'border-b-ink font-semibold text-ink no-underline hover:text-ink hover:no-underline']">
        {{ $t('index.summary') }}
      </RouterLink>
      <RouterLink :to="repoLocation('repo-commits', repoPath)" :class="['relative -mb-px inline-block cursor-pointer border-b-2 border-transparent px-3 py-1.5 text-[100%] text-ink-muted no-underline hover:text-ink hover:no-underline', active('log') && 'border-b-ink font-semibold text-ink no-underline hover:text-ink hover:no-underline']">
        {{ $t('index.log') }}
      </RouterLink>
      <RouterLink :to="repoLocation('repo-refs', repoPath)" :class="['relative -mb-px inline-block cursor-pointer border-b-2 border-transparent px-3 py-1.5 text-[100%] text-ink-muted no-underline hover:text-ink hover:no-underline', active('refs') && 'border-b-ink font-semibold text-ink no-underline hover:text-ink hover:no-underline']">
        {{ $t('nav.refs') }}
      </RouterLink>
      <RouterLink
        v-if="issuesEnabled"
        :to="repoLocation('repo-issues', repoPath)"
        :class="['relative -mb-px inline-block cursor-pointer border-b-2 border-transparent px-3 py-1.5 text-[100%] text-ink-muted no-underline hover:text-ink hover:no-underline', active('issues') && 'border-b-ink font-semibold text-ink no-underline hover:text-ink hover:no-underline']"
      >
        {{ $t('nav.issues') }}
      </RouterLink>
      <RouterLink :to="repoLocation('repo-pulls', repoPath)" :class="['relative -mb-px inline-block cursor-pointer border-b-2 border-transparent px-3 py-1.5 text-[100%] text-ink-muted no-underline hover:text-ink hover:no-underline', active('pulls') && 'border-b-ink font-semibold text-ink no-underline hover:text-ink hover:no-underline']">
        {{ $t('nav.pulls') }}
      </RouterLink>
      <RouterLink
        v-if="canAdmin"
        :to="repoLocation('repo-settings', repoPath)"
        :class="['relative -mb-px inline-block cursor-pointer border-b-2 border-transparent px-3 py-1.5 text-[100%] text-ink-muted no-underline hover:text-ink hover:no-underline', active('settings') && 'border-b-ink font-semibold text-ink no-underline hover:text-ink hover:no-underline']"
      >
        {{ $t('nav.settings') }}
      </RouterLink>
    </nav>

    <div class="pt-1.5">
      <slot v-if="repo" />
    </div>
  </div>
</template>
