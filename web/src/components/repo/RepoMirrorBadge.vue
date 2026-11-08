<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { reposApi } from '@/api'
import type { MirrorStatus, Repository } from '@/api/types'
import { useAnchoredMenu } from '@/composables/useAnchoredMenu'
import { hasMirror, mirrorRemoteLabel, mirrorUiStatus, mirrorWebURL } from '@/lib/mirror'
import Icon from '@/components/ui/Icon.vue'
import { formatDate, i18n } from '@/i18n'

const props = defineProps<{
  repoPath: string
  repo: Repository | null
}>()

const route = useRoute()
const mirrorOpen = ref(false)
const mirrorRoot = ref<HTMLElement | null>(null)
const mirrorMenu = ref<HTMLElement | null>(null)
const liveStatus = ref<MirrorStatus | null>(null)

const { style: mirrorMenuStyle, place: placeMirrorMenu } = useAnchoredMenu({
  open: mirrorOpen,
  anchor: mirrorRoot,
  menu: mirrorMenu,
  width: '16rem',
  align: 'start',
})

const mirrorDirection = computed(() => props.repo?.mirror_direction || '')
const mirrorURL = computed(() => (props.repo?.mirror_url || '').trim())
const mirrorLinkLabel = computed(() => mirrorRemoteLabel(mirrorURL.value))
const mirrorHref = computed(() => mirrorWebURL(mirrorURL.value))
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

  const target = ev.target
  if (mirrorRoot.value?.contains(target) || mirrorMenu.value?.contains(target)) {
    return
  }

  closeMirrorPopover()
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
</script>

<template>
  <span
    v-if="isMirror"
    ref="mirrorRoot"
    class="relative z-30 ml-1.5 inline-flex align-middle"
  >
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
    <Teleport to="body">
      <div
        v-if="mirrorOpen"
        ref="mirrorMenu"
        role="dialog"
        :aria-label="$t('repo.mirrorPopoverTitle')"
        class="glass-menu p-3 text-[10pt] font-sans normal-case"
        :style="mirrorMenuStyle"
        @vue:mounted="placeMirrorMenu"
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
    </Teleport>
  </span>
</template>
