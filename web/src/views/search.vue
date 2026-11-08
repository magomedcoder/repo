<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { searchApi } from '@/api'
import type { CodeSearchHit } from '@/api/search'
import type { Repository } from '@/api/types'
import PageState from '@/components/ui/PageState.vue'
import { localizeError } from '@/i18n'
import { repoLocation } from '@/lib/repoNav'
import { repositoryPath } from '@/lib/repoPath'
import Icon from '@/components/ui/Icon.vue'

type Tab = 'repositories' | 'code'

const route = useRoute()
const router = useRouter()
const q = ref(String(route.query.q || ''))
const tab = ref<Tab>(route.query.type === 'code' ? 'code' : 'repositories')
const PAGE_SIZE = 50

const repos = ref<Repository[]>([])
const codeHits = ref<CodeSearchHit[]>([])
const codeCursor = ref('')
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(false)
const error = ref('')
let timer: ReturnType<typeof setTimeout> | null = null
let abort: AbortController | null = null

const canSearch = computed(() => q.value.trim().length >= 2)

function tabClass(id: Tab) {
  return tab.value === id
    ? 'border-ink text-ink'
    : 'border-transparent text-ink-muted hover:text-ink'
}

async function runSearch(append = false) {
  const query = q.value.trim()
  if (query.length < 2) {
    repos.value = []
    codeHits.value = []
    codeCursor.value = ''
    hasMore.value = false
    error.value = ''
    return
  }

  if (!append) {
    abort?.abort()
    abort = new AbortController()
    codeCursor.value = ''
  }
  const signal = abort?.signal

  if (append) {
    loadingMore.value = true
  } else {
    loading.value = true
  }

  error.value = ''
  try {
    if (tab.value === 'code') {
      const res = await searchApi.code(
        query,
        append && codeCursor.value ? { cursor: codeCursor.value } : undefined,
        signal ? { signal } : undefined,
      )
      
      if (signal?.aborted) {
        return
      }

      const batch = res.items ?? []
      codeHits.value = append ? [...codeHits.value, ...batch] : batch
      codeCursor.value = res.next_cursor || ''
      repos.value = []
      hasMore.value = !!res.next_cursor
    } else {
      const offset = append ? repos.value.length : 0
      const res = await searchApi.repositories(
        query,
        { offset, limit: PAGE_SIZE },
        signal ? { signal } : undefined,
      )
      if (signal?.aborted) {
        return
      }
      const batch = res.repos ?? []
      repos.value = append ? [...repos.value, ...batch] : batch
      codeHits.value = []
      hasMore.value = batch.length >= PAGE_SIZE
    }
  } catch (err) {
    if (signal?.aborted) {
      return
    }
    error.value = localizeError(err, 'errors.searchFailed')
  } finally {
    if (!signal?.aborted) {
      loading.value = false
      loadingMore.value = false
    }
  }
}

function loadMore() {
  if (!loadingMore.value && hasMore.value && canSearch.value) {
    void runSearch(true)
  }
}

function schedule() {
  if (timer) {
    clearTimeout(timer)
  }

  timer = setTimeout(() => {
    void router.replace({
      name: 'search',
      query: {
        q: q.value.trim() || undefined,
        type: tab.value === 'code' ? 'code' : undefined,
      },
    })
    void runSearch(false)
  }, 250)
}

function setTab(next: Tab) {
  if (tab.value === next) {
    return
  }
  tab.value = next
  schedule()
}

function repoTo(repo: Repository) {
  return repoLocation('repo', repositoryPath(repo))
}

function codeTo(hit: CodeSearchHit) {
  return repoLocation('repo-blob', hit.repo_path || hit.repo_name, {
    query: {
      path: hit.path,
      ref: 'HEAD',
    },
  })
}

function highlight(content: string, query: string) {
  const needle = query.trim()
  if (!needle) {
    return [{ text: content, mark: false }]
  }
  const lower = content.toLowerCase()
  const n = needle.toLowerCase()
  const parts: { text: string; mark: boolean }[] = []
  let i = 0
  while (i < content.length) {
    const at = lower.indexOf(n, i)
    if (at < 0) {
      parts.push({ text: content.slice(i), mark: false })
      break
    }
    if (at > i) {
      parts.push({ text: content.slice(i, at), mark: false })
    }
    parts.push({ text: content.slice(at, at + needle.length), mark: true })
    i = at + needle.length
  }
  return parts.length ? parts : [{ text: content, mark: false }]
}

watch(() => route.query.q, (v) => {
  const next = typeof v === 'string' ? v : ''
  if (next !== q.value) {
    q.value = next
  }
})

watch(() => route.query.type, (v) => {
  const next: Tab = v === 'code' ? 'code' : 'repositories'
  if (next !== tab.value) {
    tab.value = next
  }
})

onMounted(() => {
  if (canSearch.value) {
    void runSearch(false)
  }
})
onUnmounted(() => {
  if (timer) clearTimeout(timer)
  abort?.abort()
})
</script>

<template>
  <div class="page-enter space-y-4">
    <header class="space-y-3">
      <h1 class="text-[1.05rem] font-semibold leading-snug text-ink">{{ $t('search.title') }}</h1>
      <div class="surface p-2.5 sm:p-3">
        <label class="sr-only" for="search-page-q">{{ $t('search.query') }}</label>
        <div class="search-field w-full">
          <Icon name="search" size="sm" class="shrink-0" />
          <input
            id="search-page-q"
            v-model="q"
            type="search"
            class="w-full min-w-0"
            :placeholder="$t('search.placeholder')"
            @input="schedule"
          >
        </div>
      </div>
      <nav class="flex gap-4 border-b border-line-strong/40 text-[13px]" aria-label="search type">
        <button
          type="button"
          class="border-b-2 px-0.5 pb-2 font-medium transition-colors"
          :class="tabClass('repositories')"
          @click="setTab('repositories')"
        >
          {{ $t('search.tabRepos') }}
          <span v-if="tab === 'repositories' && repos.length" class="ml-1 text-ink-muted">{{ repos.length }}{{ hasMore ? '+' : '' }}</span>
        </button>
        <button
          type="button"
          class="border-b-2 px-0.5 pb-2 font-medium transition-colors"
          :class="tabClass('code')"
          @click="setTab('code')"
        >
          {{ $t('search.tabCode') }}
          <span v-if="tab === 'code' && codeHits.length" class="ml-1 text-ink-muted">{{ codeHits.length }}</span>
        </button>
      </nav>
    </header>

    <p v-if="q.trim() && !canSearch" class="text-sm text-ink-muted">{{ $t('search.queryTooShort') }}</p>

    <PageState :loading="loading" :error="error">
      <template v-if="tab === 'repositories'">
        <ul v-if="repos.length" class="surface divide-y divide-line-strong/30 overflow-hidden">
          <li v-for="repo in repos" :key="repo.id">
            <RouterLink
              :to="repoTo(repo)"
              class="flex items-start gap-2 px-3 py-3 text-ink no-underline transition-colors hover:bg-row-hover sm:px-4"
            >
              <span class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full border border-line-strong/70 bg-white/40 text-ink-muted">
                <Icon name="repo" size="md" />
              </span>
              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-center gap-x-1.5 gap-y-0.5">
                  <span class="text-[14px] font-semibold text-link">{{ repo.name }}</span>
                  <span
                    v-if="repo.is_private"
                    class="inline-flex items-center rounded-full border border-line-strong/70 px-1.5 py-px text-[10px] font-medium text-ink-muted"
                  >
                    {{ $t('home.private') }}
                  </span>
                </div>
                <p v-if="repo.description" class="mt-0.5 line-clamp-2 text-[12px] text-ink-muted">
                  {{ repo.description }}
                </p>
                <p v-if="repo.owner" class="mt-1 text-[11px] text-ink-muted">{{ repo.owner }}</p>
              </div>
            </RouterLink>
          </li>
        </ul>
        <p v-else-if="canSearch" class="text-sm text-ink-muted">{{ $t('search.empty') }}</p>
        <p v-if="hasMore && repos.length" class="mt-3">
          <button type="button" class="btn" :disabled="loadingMore" @click="loadMore">
            {{ loadingMore ? $t('common.loading') : $t('common.loadMore') }}
          </button>
        </p>
      </template>

      <template v-else>
        <ul v-if="codeHits.length" class="space-y-3">
          <li
            v-for="(hit, i) in codeHits"
            :key="`${hit.repo_path}:${hit.path}:${hit.line}:${i}`"
            class="surface overflow-hidden"
          >
            <div class="flex flex-wrap items-baseline gap-x-2 gap-y-0.5 border-b border-line-strong/35 px-3 py-2 sm:px-4">
              <RouterLink
                :to="codeTo(hit)"
                class="inline-flex min-w-0 items-center gap-1.5 font-mono text-[12px] font-semibold text-link no-underline hover:underline"
              >
                <Icon name="repo" size="sm" class="shrink-0 text-ink-muted" />
                <span class="truncate">{{ hit.path }}</span>
              </RouterLink>
              <span class="text-[11px] text-ink-muted">
                {{ $t('search.codeIn', { repo: hit.repo_name }) }}
              </span>
            </div>
            <RouterLink
              :to="codeTo(hit)"
              class="block px-3 py-2 font-mono text-[12px] leading-relaxed text-ink no-underline hover:bg-row-hover sm:px-4"
            >
              <span class="mr-3 inline-block w-8 shrink-0 select-none text-right text-ink-muted">{{ hit.line }}</span>
              <span>
                <template v-for="(part, pi) in highlight(hit.content, q)" :key="pi">
                  <mark v-if="part.mark" class="rounded-sm bg-warn/25 text-ink">{{ part.text }}</mark>
                  <template v-else>{{ part.text }}</template>
                </template>
              </span>
            </RouterLink>
          </li>
        </ul>
        <p v-else-if="canSearch" class="text-sm text-ink-muted">{{ $t('search.empty') }}</p>
      </template>
    </PageState>
  </div>
</template>
