<script setup lang="ts">
import { btnGhost, dataTable } from '@/lib/classes'
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { reposApi } from '@/api'
import type { BlobContent, BlameResult } from '@/api/types'
import RepoShell from '@/components/repo/RepoShell.vue'
import PageState from '@/components/ui/PageState.vue'
import type { PageError } from '@/i18n'
import { i18n, localizeError, pageError } from '@/i18n'

const props = defineProps<{
  repoPath: string
}>()

const route = useRoute()
const blob = ref<BlobContent | null>(null)
const blame = ref<BlameResult | null>(null)
const error = ref<PageError | string>('')
const blameError = ref('')
const loading = ref(true)
const blameLoading = ref(false)
const showBlame = ref(false)

const LINE_HEIGHT = 18
const VIRT_THRESHOLD = 200
const WINDOW_OVERSCAN = 40

const codeScrollEl = ref<HTMLElement | null>(null)
const scrollTop = ref(0)
const viewportHeight = ref(480)

const textLines = computed(() => (blob.value?.content || '').split('\n'))
const virtualize = computed(() => !showBlame.value && textLines.value.length > VIRT_THRESHOLD)

const visibleStart = computed(() => {
  if (!virtualize.value) {
    return 0
  }

  return Math.max(0, Math.floor(scrollTop.value / LINE_HEIGHT) - WINDOW_OVERSCAN)
})

const visibleEnd = computed(() => {
  if (!virtualize.value) {
    return textLines.value.length
  }

  const end = Math.ceil((scrollTop.value + viewportHeight.value) / LINE_HEIGHT) + WINDOW_OVERSCAN
  return Math.min(textLines.value.length, end)
})

const topSpacer = computed(() => visibleStart.value * LINE_HEIGHT)
const bottomSpacer = computed(() => Math.max(0, (textLines.value.length - visibleEnd.value) * LINE_HEIGHT))

const visibleLineIndices = computed(() => {
  const out: number[] = []
  for (let i = visibleStart.value; i < visibleEnd.value; i += 1) {
    out.push(i)
  }

  return out
})

let resizeObserver: ResizeObserver | null = null

function onCodeScroll(event: Event) {
  scrollTop.value = (event.target as HTMLElement).scrollTop
}

function attachCodeViewport() {
  resizeObserver?.disconnect()
  const el = codeScrollEl.value
  if (!el) {
    return
  }
  
  viewportHeight.value = el.clientHeight || viewportHeight.value
  scrollTop.value = el.scrollTop
  resizeObserver = new ResizeObserver(() => {
    if (codeScrollEl.value) {
      viewportHeight.value = codeScrollEl.value.clientHeight || viewportHeight.value
    }
  })
  resizeObserver.observe(el)
}

onMounted(() => {
  nextTick(attachCodeViewport)
})

onUnmounted(() => {
  resizeObserver?.disconnect()
})

const refName = computed(() => (route.query.ref as string) || 'main')
const filePath = computed(() => (route.query.path as string) || '')
const parentDir = computed(() => {
  const parts = filePath.value.split('/')
  parts.pop()
  return parts.join('/')
})

const rawFilename = computed(() => {
  const parts = filePath.value.split('/')
  return parts[parts.length - 1] || i18n.global.t('blob.fallbackName')
})

const rawHref = computed(() => `/api/repos/${props.repoPath}/raw?ref=${encodeURIComponent(refName.value)}&path=${encodeURIComponent(filePath.value)}`)

function lineClass(line: string, filename: string) {
  const ext = filename.split('.').pop()?.toLowerCase() || ''
  const comment = ext === 'py' || ext === 'sh' ? /^\s*#/ : /^\s*(\/\/|\/\*|\*)/
  if (comment.test(line)) {
    return 'text-ink-muted'
  }

  if (/^\s*(func|class|def|import|package|type|const|let|var|return)\b/.test(line)) {
    return 'text-accent'
  }

  return ''
}

function shortSha(sha: string) {
  return sha.slice(0, 7)
}

async function loadBlob() {
  loading.value = true
  error.value = ''
  blame.value = null
  try {
    if (!filePath.value) {
      error.value = localizeError(new Error('path'), 'errors.pathRequired')
      return
    }
    blob.value = await reposApi.blob(props.repoPath, {
      ref: refName.value,
      path: filePath.value,
    })
  } catch (err) {
    error.value = pageError(err, 'errors.loadFile')
  } finally {
    loading.value = false
  }
}

async function loadBlame() {
  if (!filePath.value || blob.value?.is_binary) {
    return
  }
  
  blameLoading.value = true
  blameError.value = ''
  try {
    blame.value = await reposApi.blame(props.repoPath, {
      ref: refName.value,
      path: filePath.value,
    })
  } catch (err) {
    blameError.value = localizeError(err, 'errors.loadBlame')
  } finally {
    blameLoading.value = false
  }
}

async function toggleBlame() {
  showBlame.value = !showBlame.value
  if (showBlame.value && !blame.value && !blameLoading.value) {
    await loadBlame()
  }
}

watch(
  () => [props.repoPath, route.query.ref, route.query.path],
  () => {
    showBlame.value = false
    loadBlob()
  },
  { immediate: true },
)

watch([blob, virtualize], () => {
  scrollTop.value = 0
  if (codeScrollEl.value) {
    codeScrollEl.value.scrollTop = 0
  }
  nextTick(attachCodeViewport)
})
</script>

<template>
  <RepoShell
    :repo-path="repoPath"
    tab="code"
  >
    <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
      <p class="min-w-0 truncate text-sm">
        <span class="text-ink-muted">{{ refName }} /</span> {{ filePath }}
      </p>
      <div class="flex gap-2">
        <button
          v-if="blob && !blob.is_binary"
          type="button"
          :class="[btnGhost, showBlame && 'bg-paper-2']"
          @click="toggleBlame"
        >
          {{ showBlame ? $t('blob.hideBlame') : $t('blob.blame') }}
        </button>
        <RouterLink
          :class="btnGhost"
          :to="{
            name: 'repo-tree',
            params: { repoPath },
            query: {
              ref: refName,
              path: parentDir || undefined
            }
          }"
        >
          {{ $t('blob.tree') }}
        </RouterLink>
        <a
          :class="btnGhost"
          :href="rawHref"
          :download="rawFilename"
          target="_blank"
          rel="noreferrer"
        >
          {{ $t('blob.raw') }}
        </a>
      </div>
    </div>

    <PageState :loading="loading" :error="error">
      <div v-if="blob && showBlame && !blob.is_binary" class="term-block">
        <p v-if="blameLoading" class="p-4 text-sm text-ink-muted">{{ $t('common.loading') }}</p>
        <p v-else-if="blameError" class="p-4 text-sm text-warn">{{ blameError }}</p>
        <div v-else-if="blame" class="overflow-x-auto">
          <table class="w-full font-mono text-xs">
            <tbody>
              <tr
                v-for="line in blame.lines"
                :key="line.line"
                class="border-b border-line/60 hover:bg-paper"
              >
                <td class="whitespace-nowrap px-2 py-0.5 text-right text-ink-muted">{{ line.line }}</td>
                <td class="whitespace-nowrap px-2 py-0.5 text-ink-muted">{{ shortSha(line.commit_sha) }}</td>
                <td class="whitespace-nowrap px-2 py-0.5 text-ink-muted">{{ line.author_name }}</td>
                <td class="px-2 py-0.5">{{ line.content }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
      <div v-else-if="blob" class="term-block">
        <div class="flex items-center justify-between term-block-hd text-xs text-ink-muted">
          <span>{{ $t('common.bytes', { n: blob.size }) }}</span>
          <span>{{ blob.is_binary ? $t('common.binary') : blob.encoding }}</span>
        </div>
        <div
          v-if="!blob.is_binary && virtualize"
          ref="codeScrollEl"
          class="max-h-[70vh] overflow-y-auto overflow-x-auto"
          @scroll="onCodeScroll"
        >
          <table :class="[dataTable, 'font-mono']">
            <tbody>
              <tr v-if="topSpacer" aria-hidden="true">
                <td :style="{ height: `${topSpacer}px`, padding: 0, border: 'none' }" colspan="2" />
              </tr>
              <tr v-for="i in visibleLineIndices" :key="i">
                <td class="w-10 text-right text-ink-muted">{{ i + 1 }}</td>
                <td :class="lineClass(textLines[i] ?? '', rawFilename)">{{ textLines[i] || ' ' }}</td>
              </tr>
              <tr v-if="bottomSpacer" aria-hidden="true">
                <td :style="{ height: `${bottomSpacer}px`, padding: 0, border: 'none' }" colspan="2" />
              </tr>
            </tbody>
          </table>
        </div>
        <table v-else-if="!blob.is_binary" :class="[dataTable, 'font-mono']">
          <tbody>
            <tr v-for="(line, i) in textLines" :key="i">
              <td class="w-10 text-right text-ink-muted">{{ i + 1 }}</td>
              <td :class="lineClass(line, rawFilename)">{{ line || ' ' }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else class="p-4 text-sm text-ink-muted">{{ $t('blob.binary') }}</p>
      </div>
    </PageState>
  </RepoShell>
</template>
