<script setup lang="ts">
import { onUnmounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { reposApi } from '@/api'
import type { CommitInfo, Repository } from '@/api/types'
import { useBreadcrumbs, type Crumb } from '@/composables/useBreadcrumbs'
import { formatDate, localizeError } from '@/i18n'

const props = defineProps<{
  owner: string
  repoPath: string
}>()

const { setBreadcrumbs, clearBreadcrumbs } = useBreadcrumbs()

const repo = ref<Repository | null>(null)
const commits = ref<CommitInfo[]>([])
const error = ref('')
const loading = ref(true)
const offset = ref(0)
const limit = 30

function setCrumbs() {
  const crumbs: Crumb[] = [{ label: props.owner, to: '/' }]
  props.repoPath.split('/').forEach((part, i, arr) => {
    crumbs.push(i === arr.length - 1
      ? {
          label: part,
          to: {
            name: 'repo',
            params: {
              owner: props.owner,
              repoPath: props.repoPath
            }
          }
        }
      : { label: part })
  })
  crumbs.push({ label: 'commits', labelKey: 'nav.commits' })
  setBreadcrumbs(crumbs as never)
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    repo.value = await reposApi.get(props.owner, props.repoPath)
    setCrumbs()
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

function shortSha(sha: string) {
  return sha.slice(0, 7)
}

function firstLine(message: string) {
  return message.split('\n')[0]
}

watch(() => [props.owner, props.repoPath], () => {
  offset.value = 0
  load()
}, { immediate: true })
onUnmounted(clearBreadcrumbs)
</script>

<template>
  <div>
    <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
      <h1 class="font-display text-2xl font-bold">{{ $t('commits.title') }}</h1>
      <RouterLink class="btn-ghost" :to="{ name: 'repo', params: { owner, repoPath } }">{{ $t('repo.home') }}</RouterLink>
    </div>

    <p v-if="loading" class="text-sm text-ink-muted">{{ $t('common.loading') }}</p>
    <p v-else-if="error" class="text-sm text-warn">{{ error }}</p>
    <ul v-else class="divide-y divide-line overflow-hidden rounded-lg border border-line bg-white/80">
      <li v-for="c in commits" :key="c.sha">
        <RouterLink
          :to="{
            name: 'repo-commit',
            params: {
              owner,
              repoPath,
              sha: c.sha
            }
          }"
          class="block px-4 py-3 hover:bg-moss-soft/40"
        >
          <p class="font-semibold">{{ firstLine(c.message) }}</p>
          <p class="mt-1 font-mono text-xs text-ink-muted">
            {{ shortSha(c.sha) }} {{ c.author_name }} {{ formatDate(c.authored_at) }}
          </p>
        </RouterLink>
      </li>
      <li v-if="!commits.length" class="px-4 py-8 text-center text-sm text-ink-muted">{{ $t('commits.empty') }}</li>
    </ul>

    <div v-if="commits.length === limit || offset > 0" class="mt-4 flex gap-2">
      <button
        type="button"
        class="btn-ghost"
        :disabled="offset === 0"
        @click="offset = Math.max(0, offset - limit); load()"
      >
        {{ $t('commits.newer') }}
      </button>
      <button
        type="button"
        class="btn-ghost"
        :disabled="commits.length < limit"
        @click="offset += limit; load()"
      >
        {{ $t('commits.older') }}
      </button>
    </div>
  </div>
</template>
