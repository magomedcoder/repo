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
import { useAuth } from '@/composables/useAuth'
import { formatDate, localizeError } from '@/i18n'

const props = defineProps<{
  owner: string
  repoPath: string
  number: string
}>()

const auth = useAuth()
const router = useRouter()
const pull = ref<PullDetail | null>(null)
const compare = ref<PullCompare | null>(null)
const comment = ref('')
const strategy = ref<'merge' | 'ff-only'>('merge')
const error = ref('')
const loading = ref(true)
const busy = ref(false)

const isOwner = computed(() => auth.user.value?.username === props.owner)
const canEdit = computed(() => isOwner.value || auth.user.value?.username === pull.value?.author)

function canEditComment(author: string) {
  return isOwner.value || auth.user.value?.username === author
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const detail = await pullsApi.get(props.owner, props.repoPath, Number(props.number))
    pull.value = detail
    if (detail.state === 'open') {
      compare.value = await pullsApi.diff(props.owner, props.repoPath, detail.number)
      strategy.value = compare.value.can_fast_forward ? 'ff-only' : 'merge'
    } else {
      compare.value = null
    }
  } catch (err) {
    error.value = localizeError(err, 'errors.loadPull')
  } finally {
    loading.value = false
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
    await pullsApi.update(props.owner, props.repoPath, pull.value.number, { state: next })
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
    await pullsApi.merge(props.owner, props.repoPath, pull.value.number, strategy.value)
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
    await pullsApi.comment(props.owner, props.repoPath, pull.value.number, comment.value)
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
    await pullsApi.updateComment(props.owner, props.repoPath, pull.value.number, id, body)
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
    await pullsApi.removeComment(props.owner, props.repoPath, pull.value.number, id)
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.deleteFailed')
  } finally {
    busy.value = false
  }
}

async function remove() {
  if (!pull.value || !confirm(`#${pull.value.number}`)) {
    return
  }

  busy.value = true
  try {
    await pullsApi.remove(props.owner, props.repoPath, pull.value.number)
    await router.push({ 
      name: 'repo-pulls', 
      params: { 
        owner: props.owner, 
        repoPath: props.repoPath 
      } 
    })
  } catch (err) {
    error.value = localizeError(err, 'errors.deleteFailed')
    busy.value = false
  }
}

watch(() => [props.owner, props.repoPath, props.number], load, { immediate: true })
</script>

<template>
  <RepoShell 
    :owner="owner" 
    :repo-path="repoPath" 
    tab="pulls"
  >
    <PageState :loading="loading" :error="error">
      <template v-if="pull">
        <div class="mb-4 flex flex-wrap items-start justify-between gap-3">
          <div>
            <h1 class="text-2xl font-semibold">
              {{ pull.title }}
              <span class="font-normal text-ink-muted">#{{ pull.number }}</span>
            </h1>
            <p class="mt-1 font-mono text-sm text-ink-muted">{{ pull.head_branch }} -> {{ pull.base_branch }}</p>
          </div>
          <div class="flex flex-wrap gap-2">
            <button 
              v-if="canEdit && pull.state !== 'merged'" 
              type="button" 
              class="btn-ghost" 
              :disabled="busy" 
              @click="toggleState"
            >
              {{ pull.state === 'open' ? $t('pulls.close') : $t('pulls.reopen') }}
            </button>
            <button 
              v-if="canEdit" 
              type="button" 
              class="btn-danger" 
              :disabled="busy" 
              @click="remove"
            >{{ $t('pulls.delete') }}</button>
          </div>
        </div>

        <div class="mb-4 flex flex-wrap items-center gap-2 text-sm">
          <span
            class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-semibold text-white"
            :class="pull.state === 'open' ? 'bg-open' : pull.state === 'merged' ? 'bg-done' : 'bg-ink-muted'"
          >
            <Icon :name="pull.state === 'merged' ? 'pull-merged' : 'pull'" />
            {{ $t(`pulls.${pull.state}`) }}
          </span>
          <span class="text-ink-muted">{{ pull.author }} {{ formatDate(pull.created_at) }}</span>
          <span v-if="pull.merged_by" class="text-ink-muted"> {{ $t('pulls.mergedBy', { user: pull.merged_by }) }}</span>
        </div>

        <article class="mb-4 overflow-hidden rounded-md border border-line bg-white">
          <header class="border-b border-line bg-paper px-4 py-2 text-xs">
            <span class="font-semibold">{{ pull.author }}</span>
            <span class="text-ink-muted"> {{ formatDate(pull.created_at) }}</span>
          </header>
          <p class="whitespace-pre-wrap px-4 py-3 text-sm">{{ pull.body || $t('pulls.noBody') }}</p>
        </article>

        <form
          v-if="isOwner && pull.state === 'open'"
          class="mb-4 flex flex-wrap items-end gap-3 rounded-md border border-line bg-white p-4"
          @submit.prevent="merge"
        >
          <label class="block text-sm">
            <span class="mb-1 block font-semibold">{{ $t('pulls.strategy') }}</span>
            <select v-model="strategy" class="input min-w-40">
              <option value="merge">{{ $t('pulls.strategyMerge') }}</option>
              <option value="ff-only" :disabled="!!compare && !compare.can_fast_forward">{{ $t('pulls.strategyFf') }}</option>
            </select>
          </label>
          <button 
            type="submit" 
            class="btn-primary" 
            :disabled="busy"
          >{{ $t('pulls.merge') }}</button>
          <p v-if="compare?.can_fast_forward" class="text-xs text-open">{{ $t('pulls.canFf') }}</p>
        </form>

        <template v-if="compare">
          <h2 class="mb-2 text-sm font-semibold">{{ $t('pulls.commitsHeading', { n: compare.commits.length }) }}</h2>
          <div class="mb-4">
            <CommitList 
              :commits="compare.commits" 
              :owner="owner" 
              :repo-path="repoPath" 
            />
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
        <form class="space-y-2" @submit.prevent="addComment">
          <textarea 
            v-model="comment" 
            class="input min-h-24" 
            required 
            :placeholder="$t('pulls.comment')" 
          />
          <button 
            type="submit" 
            class="btn-primary" 
            :disabled="busy"
          >{{ $t('pulls.commentAction') }}</button>
        </form>
      </template>
    </PageState>
  </RepoShell>
</template>
