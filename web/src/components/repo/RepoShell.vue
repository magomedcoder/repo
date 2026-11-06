<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { reposApi } from '@/api'
import type { MirrorStatus, Repository } from '@/api/types'
import { useCanWrite } from '@/composables/useCanWrite'
import { hasMirror, mirrorRemoteLabel, mirrorUiStatus, mirrorWebURL } from '@/lib/mirror'
import { folderLocation } from '@/lib/folderNav'
import { repoLocation } from '@/lib/repoNav'
import { normalizeRepoPath } from '@/lib/repoPath'
import Icon from '@/components/ui/Icon.vue'
import PageState from '@/components/ui/PageState.vue'
import type { PageError } from '@/i18n'
import { formatDate, i18n } from '@/i18n'

const props = defineProps<{
  repoPath: string
  tab: 'summary' | 'log' | 'commit' | 'refs' | 'issues' | 'pulls' | 'settings'
  repo: Repository | null
  loading?: boolean
  error?: PageError | string
}>()

const route = useRoute()
const { canAdmin } = useCanWrite(() => props.repo)
const mirrorOpen = ref(false)
const mirrorRoot = ref<HTMLElement | null>(null)

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
const mirrorLinkLabel = computed(() => mirrorRemoteLabel(mirrorURL.value))
const mirrorHref = computed(() => mirrorWebURL(mirrorURL.value))
const liveStatus = ref<MirrorStatus | null>(null)
const mirrorState = computed(() => mirrorUiStatus(props.repo ? {
  mirror_direction: props.repo.mirror_direction,
  mirror_enabled: props.repo.mirror_enabled,
  mirror_status: liveStatus.value?.status ?? props.repo.mirror_status,
  mirror_last_error: liveStatus.value ? liveStatus.value.last_error : props.repo.mirror_last_error,
  mirror_last_sync_at: liveStatus.value?.last_sync_at ?? props.repo.mirror_last_sync_at,
} : null))
const mirrorLastError = computed(() => (liveStatus.value ? liveStatus.value.last_error : props.repo?.mirror_last_error) || '')
const mirrorLastSyncAt = computed(() => (liveStatus.value?.last_sync_at ?? props.repo?.mirror_last_sync_at) || '')
const isMirror = computed(() => mirrorDirection.value === 'pull' || mirrorDirection.value === 'push')
const mirrorDirectionLabel = computed(() => {
  if (mirrorDirection.value === 'pull') {
    return i18n.global.t('settings.mirrorPull')
  }
  if (mirrorDirection.value === 'push') {
    return i18n.global.t('settings.mirrorPush')
  }
  return ''
})
const mirrorStatusLabel = computed(() => (
  mirrorState.value ? i18n.global.t(`mirror.status.${mirrorState.value}`) : ''
))
const mirrorAriaLabel = computed(() => {
  if (!isMirror.value) {
    return ''
  }

  const direction = mirrorDirection.value === 'pull'
    ? i18n.global.t('repo.mirrorPullPrefix')
    : i18n.global.t('repo.mirrorPushPrefix')
  const remote = mirrorLinkLabel.value || i18n.global.t('repo.mirrorRemoteUnknown')
  return mirrorStatusLabel.value
    ? i18n.global.t('repo.mirrorTooltip', { direction, remote, status: mirrorStatusLabel.value })
    : `${direction}${remote}`
})

let pollTimer: ReturnType<typeof setInterval> | undefined

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = undefined
  }
}

function toggleMirrorPopover() {
  mirrorOpen.value = !mirrorOpen.value
}

function closeMirrorPopover() {
  mirrorOpen.value = false
}

function onDocPointerDown(ev: PointerEvent) {
  if (!mirrorOpen.value || !(ev.target instanceof Node)) {
    return
  }
  
  if (mirrorRoot.value && !mirrorRoot.value.contains(ev.target)) {
    closeMirrorPopover()
  }
}

function onDocKeydown(ev: KeyboardEvent) {
  if (ev.key === 'Escape') {
    closeMirrorPopover()
  }
}

watch(
  () => [props.repoPath, props.repo?.mirror_status, hasMirror(props.repo)] as const,
  () => {
    liveStatus.value = null
    closeMirrorPopover()
  },
)

watch(
  () => route.fullPath,
  () => {
    closeMirrorPopover()
  },
)

watch(
  () => mirrorState.value === 'syncing',
  (syncing) => {
    stopPolling()
    if (!syncing) {
      return
    }

    pollTimer = setInterval(async () => {
      try {
        liveStatus.value = await reposApi.mirrorStatus(props.repoPath)
      } catch {
        stopPolling()
      }
    }, 4000)
  },
  { immediate: true },
)

onMounted(() => {
  document.addEventListener('pointerdown', onDocPointerDown)
  document.addEventListener('keydown', onDocKeydown)
})

onBeforeUnmount(() => {
  stopPolling()
  document.removeEventListener('pointerdown', onDocPointerDown)
  document.removeEventListener('keydown', onDocKeydown)
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
      <span
        v-if="repo?.is_private"
        class="ml-1.5 inline-flex align-middle text-ink-muted"
        :title="$t('home.private')"
        :aria-label="$t('settings.private')"
      >
        <Icon name="lock" size="sm" />
      </span>
      <span v-if="isMirror" ref="mirrorRoot" class="relative ml-1.5 inline-flex align-middle">
        <button
          type="button"
          class="inline-flex cursor-pointer items-center border-0 bg-transparent p-0 text-ink-muted hover:text-ink"
          :class="mirrorState === 'error' && 'text-warn'"
          :aria-expanded="mirrorOpen"
          aria-haspopup="dialog"
          :aria-label="mirrorAriaLabel"
          @click="toggleMirrorPopover"
        >
          <Icon name="mirror" size="sm" />
        </button>
        <div
          v-if="mirrorOpen"
          role="dialog"
          :aria-label="$t('repo.mirrorPopoverTitle')"
          class="absolute left-0 top-full z-20 mt-1 w-64 border border-line bg-paper p-3 text-[10pt] font-sans normal-case shadow-sm"
        >
          <p class="mb-2 font-semibold text-ink">{{ $t('repo.mirrorPopoverTitle') }}</p>
          <dl class="space-y-1.5 text-ink">
            <div>
              <dt class="text-[85%] text-ink-muted">{{ $t('settings.mirrorDirection') }}</dt>
              <dd>{{ mirrorDirectionLabel }}</dd>
            </div>
            <div v-if="mirrorStatusLabel">
              <dt class="text-[85%] text-ink-muted">{{ $t('repo.mirrorStatus') }}</dt>
              <dd :class="mirrorState === 'error' && 'text-warn'">{{ mirrorStatusLabel }}</dd>
            </div>
            <div>
              <dt class="text-[85%] text-ink-muted">{{ $t('repo.mirrorRemote') }}</dt>
              <dd class="break-all">
                <a
                  v-if="mirrorHref"
                  :href="mirrorHref"
                  class="text-link underline decoration-line underline-offset-2 hover:text-ink"
                  target="_blank"
                  rel="noopener noreferrer"
                >{{ mirrorLinkLabel || $t('repo.mirrorOpenRemote') }}</a>
                <span v-else-if="mirrorLinkLabel">{{ mirrorLinkLabel }}</span>
                <span v-else class="text-ink-muted">{{ $t('repo.mirrorRemoteUnknown') }}</span>
              </dd>
            </div>
            <div v-if="mirrorLastSyncAt">
              <dt class="text-[85%] text-ink-muted">{{ $t('repo.mirrorLastSyncLabel') }}</dt>
              <dd>{{ formatDate(mirrorLastSyncAt) }}</dd>
            </div>
            <div v-if="mirrorLastError">
              <dt class="text-[85%] text-ink-muted">{{ $t('mirror.status.error') }}</dt>
              <dd class="break-words text-warn">{{ mirrorLastError }}</dd>
            </div>
          </dl>
        </div>
      </span>
    </div>

    <nav class="m-0 mb-0 flex w-full flex-wrap items-stretch gap-0 border-b border-line" :aria-label="$t('nav.repo')">
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
