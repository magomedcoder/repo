<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { pullsApi } from '@/api'
import type { PullCompare, PullDetail } from '@/api/types'
import CommentCard from '@/components/issues/CommentCard.vue'
import CommitList from '@/components/repo/CommitList.vue'
import DiffFileList from '@/components/repo/DiffFileList.vue'
import RepoShell from '@/components/repo/RepoShell.vue'
import Icon from '@/components/ui/Icon.vue'
import PageState from '@/components/ui/PageState.vue'
import { useCanWrite } from '@/composables/useCanWrite'
import { useRepo } from '@/composables/useRepo'
import { formatDate, i18n, localizeError } from '@/i18n'
import { repoLocation } from '@/lib/repoNav'

const props = defineProps<{
  repoPath: string
  number: string
}>()

const router = useRouter()
const { repo, loading: repoLoading, error: repoError } = useRepo(() => props.repoPath)
const { isAuthenticated, canAdmin, username } = useCanWrite(repo)
const pull = ref<PullDetail | null>(null)
const compare = ref<PullCompare | null>(null)
const comment = ref('')
const strategy = ref<'merge' | 'ff-only'>('merge')
const error = ref('')
const dataLoading = ref(false)
const busy = ref(false)

const loading = computed(() => dataLoading.value)
const canEdit = computed(() => canAdmin.value || (!!username.value && username.value === pull.value?.author))

function canEditComment(author: string) {
  return canAdmin.value || username.value === author
}

async function load() {
  dataLoading.value = true
  error.value = ''
  try {
    const detail = await pullsApi.get(props.repoPath, Number(props.number))
    pull.value = detail
    if (detail.state === 'open') {
      compare.value = await pullsApi.diff(props.repoPath, detail.number)
      strategy.value = compare.value.can_fast_forward ? 'ff-only' : 'merge'
    } else {
      compare.value = null
    }
  } catch (err) {
    error.value = localizeError(err, 'errors.loadPull')
  } finally {
    dataLoading.value = false
  }
}

async function toggleState() {
  if (!pull.value) {
    return
  }

  busy.value = true
  error.value = ''
  try {
    const next = pull.value.state === 'open' ? 'closed' : 'open'
    await pullsApi.update(props.repoPath, pull.value.number, { state: next })
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.saveFailed')
  } finally {
    busy.value = false
  }
}

async function merge() {
  if (!pull.value) {
    return
  }

  busy.value = true
  error.value = ''
  try {
    await pullsApi.merge(props.repoPath, pull.value.number, strategy.value)
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.mergeFailed')
  } finally {
    busy.value = false
  }
}

async function addComment() {
  if (!pull.value) {
    return
  }

  busy.value = true
  error.value = ''
  try {
    await pullsApi.comment(props.repoPath, pull.value.number, comment.value)
    comment.value = ''
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.createComment')
  } finally {
    busy.value = false
  }
}

async function saveComment(id: number, body: string) {
  if (!pull.value) {
    return
  }

  busy.value = true
  error.value = ''
  try {
    await pullsApi.updateComment(props.repoPath, pull.value.number, id, body)
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.saveFailed')
  } finally {
    busy.value = false
  }
}

async function removeComment(id: number) {
  if (!pull.value) {
    return
  }

  busy.value = true
  error.value = ''
  try {
    await pullsApi.removeComment(props.repoPath, pull.value.number, id)
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.deleteFailed')
  } finally {
    busy.value = false
  }
}

async function remove() {
  if (!pull.value || !confirm(i18n.global.t('pulls.confirmDelete', { n: pull.value.number }))) return
  busy.value = true
  try {
    await pullsApi.remove(props.repoPath, pull.value.number)
    await router.push(repoLocation('repo-pulls', props.repoPath))
  } catch (err) {
    error.value = localizeError(err, 'errors.deleteFailed')
    busy.value = false
  }
}

watch([() => props.repoPath, () => props.number], load, { immediate: true })
</script>

<template>
  <RepoShell 
    :repo-path="repoPath" 
    tab="pulls" 
    :repo="repo" 
    :loading="repoLoading" 
    :error="repoError"
  >
    <PageState :loading="loading" :error="error">
      <template v-if="pull">
        <div class="mb-4 flex flex-wrap items-start justify-between gap-3">
          <div>
            <h1 class="text-[140%] font-bold text-ink">
              {{ pull.title }}
              <span class="font-normal text-ink-muted">#{{ pull.number }}</span>
            </h1>
            <p class="mt-1 text-sm text-ink-muted">{{ pull.head_branch }} -> {{ pull.base_branch }}</p>
          </div>
          <div class="flex flex-wrap gap-2">
            <button v-if="canEdit && pull.state !== 'merged'" type="button" class="inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-ink hover:bg-row-hover disabled:cursor-not-allowed disabled:opacity-50" :disabled="busy" @click="toggleState">
              {{ pull.state === 'open' ? $t('pulls.close') : $t('pulls.reopen') }}
            </button>
            <button v-if="canEdit" type="button" class="inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-warn hover:bg-warn-soft disabled:cursor-not-allowed disabled:opacity-50" :disabled="busy" @click="remove">
              {{ $t('pulls.delete') }}
            </button>
          </div>
        </div>

        <div class="mb-4 flex flex-wrap items-center gap-2 text-sm">
          <span class="inline-flex items-center gap-1 text-ink-muted">
            <Icon :name="pull.state === 'merged' ? 'pull-merged' : 'pull'" />
            {{ $t(`pulls.${pull.state}`) }}
          </span>
          <span class="text-ink-muted">{{ pull.author }} {{ formatDate(pull.created_at) }}</span>
          <span v-if="pull.merged_by" class="text-ink-muted"> {{ $t('pulls.mergedBy', { user: pull.merged_by }) }}</span>
        </div>

        <article class="overflow-hidden border border-line bg-paper mb-4">
          <header class="flex items-center gap-2 border-b border-line bg-path px-3 py-1.5 text-[10pt] text-ink text-xs">
            <span class="font-semibold">{{ pull.author }}</span>
            <span class="text-ink-muted"> {{ formatDate(pull.created_at) }}</span>
          </header>
          <p class="whitespace-pre-wrap px-4 py-3 text-sm">{{ pull.body || $t('pulls.noBody') }}</p>
        </article>

        <form v-if="canAdmin && pull.state === 'open'" class="overflow-hidden border border-line bg-paper mb-4" @submit.prevent="merge">
          <div class="flex items-center gap-2 border-b border-line bg-path px-3 py-1.5 text-[10pt] text-ink">{{ $t('pulls.merge') }}</div>
          <div class="p-3 sm:p-4 flex flex-wrap items-end gap-3">
            <label class="block text-sm">
              <span class="mb-1 block font-semibold">{{ $t('pulls.strategy') }}</span>
              <select v-model="strategy" :class="['w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none', 'min-w-40']">
                <option value="merge">{{ $t('pulls.strategyMerge') }}</option>
                <option value="ff-only" :disabled="!!compare && !compare.can_fast_forward">{{ $t('pulls.strategyFf') }}</option>
              </select>
            </label>
            <button type="submit" class="inline-flex items-center justify-center border border-ink bg-ink px-2.5 py-1 text-[10pt] text-paper hover:bg-ink-muted disabled:cursor-not-allowed disabled:opacity-50" :disabled="busy">{{ $t('pulls.merge') }}</button>
            <p v-if="compare?.can_fast_forward" class="text-xs text-open">{{ $t('pulls.canFf') }}</p>
          </div>
        </form>

        <template v-if="compare">
          <h2 class="mb-2 text-sm font-semibold">{{ $t('pulls.commitsHeading', { n: compare.commits.length }) }}</h2>
          <div class="mb-4">
            <CommitList :commits="compare.commits" :repo-path="repoPath" />
          </div>
          <h2 class="mb-2 text-sm font-semibold">{{ $t('pulls.filesHeading', { n: compare.files.length }) }}</h2>
          <div class="mb-6 space-y-3">
            <DiffFileList :files="compare.files" />
          </div>
        </template>

        <h2 class="mb-2 text-xs font-semibold uppercase tracking-wide text-ink-muted">{{ $t('pulls.discussion') }}</h2>
        <div class="mb-4 space-y-3">
          <CommentCard
            v-for="item in pull.comments"
            :key="item.id"
            :author="item.author"
            :body="item.body"
            :created-at="item.created_at"
            :can-edit="canEditComment(item.author)"
            :busy="busy"
            @save="saveComment(item.id, $event)"
            @remove="removeComment(item.id)"
          />
          <p v-if="!pull.comments.length" class="text-sm text-ink-muted">{{ $t('pulls.noComments') }}</p>
        </div>
        <form v-if="isAuthenticated" class="space-y-2" @submit.prevent="addComment">
          <textarea v-model="comment" :class="['w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none', 'min-h-24']" required :placeholder="$t('pulls.comment')" />
          <button type="submit" class="inline-flex items-center justify-center border border-ink bg-ink px-2.5 py-1 text-[10pt] text-paper hover:bg-ink-muted disabled:cursor-not-allowed disabled:opacity-50" :disabled="busy">{{ $t('pulls.commentAction') }}</button>
        </form>
      </template>
    </PageState>
  </RepoShell>
</template>
