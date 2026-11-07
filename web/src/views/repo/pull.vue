<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { pullsApi } from '@/api'
import type { PullCompare, PullDetail } from '@/api/types'
import CommentCard from '@/components/issues/CommentCard.vue'
import DiscussionDetailLayout from '@/components/issues/DiscussionDetailLayout.vue'
import CommitList from '@/components/repo/CommitList.vue'
import DiffFileList from '@/components/repo/DiffFileList.vue'
import RepoShell from '@/components/repo/RepoShell.vue'
import Icon from '@/components/ui/Icon.vue'
import PageState from '@/components/ui/PageState.vue'
import Surface from '@/components/ui/Surface.vue'
import SurfaceHeader from '@/components/ui/SurfaceHeader.vue'
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
const compareLoading = ref(false)
const compareError = ref('')
const busy = ref(false)

const loading = computed(() => dataLoading.value)
const canEdit = computed(() => canAdmin.value || (!!username.value && username.value === pull.value?.author))

function canEditComment(author: string) {
  return canAdmin.value || username.value === author
}

async function loadCompare(number: number) {
  compareLoading.value = true
  compareError.value = ''
  try {
    compare.value = await pullsApi.diff(props.repoPath, number)
    strategy.value = compare.value.can_fast_forward ? 'ff-only' : 'merge'
  } catch (err) {
    compare.value = null
    compareError.value = localizeError(err, 'errors.loadPull')
  } finally {
    compareLoading.value = false
  }
}

async function load() {
  dataLoading.value = true
  error.value = ''
  compareError.value = ''
  try {
    const detail = await pullsApi.get(props.repoPath, Number(props.number))
    pull.value = detail
    if (detail.state === 'open') {
      dataLoading.value = false
      await loadCompare(detail.number)
      return
    }
    compare.value = null
  } catch (err) {
    pull.value = null
    compare.value = null
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
      <DiscussionDetailLayout v-if="pull">
        <template #title>
          {{ pull.title }}
          <span class="font-normal text-ink-muted">#{{ pull.number }}</span>
        </template>
        <template #subtitle>
          <p class="mt-1 text-sm text-ink-muted">{{ pull.head_branch }} -> {{ pull.base_branch }}</p>
        </template>
        <template #actions>
          <button v-if="canEdit && pull.state !== 'merged'" type="button" class="btn" :disabled="busy" @click="toggleState">
            <Icon name="pull" size="sm" />
            {{ pull.state === 'open' ? $t('pulls.close') : $t('pulls.reopen') }}
          </button>
          <button v-if="canEdit" type="button" class="btn-warn" :disabled="busy" @click="remove">
            <Icon name="trash" size="sm" />
            {{ $t('pulls.delete') }}
          </button>
        </template>
        <template #meta>
          <span class="inline-flex items-center gap-1 text-ink-muted">
            <Icon :name="pull.state === 'merged' ? 'pull-merged' : 'pull'" />
            {{ $t(`pulls.${pull.state}`) }}
          </span>
          <span class="text-ink-muted">{{ pull.author }} {{ formatDate(pull.created_at) }}</span>
          <span v-if="pull.merged_by" class="text-ink-muted"> {{ $t('pulls.mergedBy', { user: pull.merged_by }) }}</span>
        </template>
        <template #body>
          <Surface as="article">
            <SurfaceHeader as="header" class="text-xs">
              <span class="font-semibold">{{ pull.author }}</span>
              <span class="text-ink-muted"> {{ formatDate(pull.created_at) }}</span>
            </SurfaceHeader>
            <p class="whitespace-pre-wrap px-4 py-3 text-sm">{{ pull.body || $t('pulls.noBody') }}</p>
          </Surface>

          <Surface
            v-if="canAdmin && pull.state === 'open'"
            as="form"
            @submit.prevent="merge"
          >
            <SurfaceHeader>{{ $t('pulls.merge') }}</SurfaceHeader>
            <div class="p-3 sm:p-4 flex flex-wrap items-end gap-3">
              <label class="block text-sm">
                <span class="mb-1 block font-semibold">{{ $t('pulls.strategy') }}</span>
                <select v-model="strategy" :class="['field', 'min-w-40']" :disabled="compareLoading">
                  <option value="merge">{{ $t('pulls.strategyMerge') }}</option>
                  <option value="ff-only" :disabled="!compare?.can_fast_forward">{{ $t('pulls.strategyFf') }}</option>
                </select>
              </label>
              <button type="submit" class="btn-primary" :disabled="busy || compareLoading">
                <Icon name="pull-merged" size="sm" />
                {{ $t('pulls.merge') }}
              </button>
              <p v-if="compare?.can_fast_forward" class="text-xs text-open">{{ $t('pulls.canFf') }}</p>
            </div>
          </Surface>

          <p v-if="compareLoading" class="text-sm text-ink-muted">{{ $t('common.loading') }}</p>
          <p v-else-if="compareError" class="text-sm text-warn">{{ compareError }}</p>
          <template v-else-if="compare">
            <div>
              <h2 class="mb-2 text-sm font-semibold">{{ $t('pulls.commitsHeading', { n: compare.commits.length }) }}</h2>
              <CommitList :commits="compare.commits" :repo-path="repoPath" />
            </div>
            <div>
              <h2 class="mb-2 text-sm font-semibold">{{ $t('pulls.filesHeading', { n: compare.files.length }) }}</h2>
              <div class="space-y-3">
                <DiffFileList :files="compare.files" />
              </div>
            </div>
          </template>
        </template>
        <template #comments>
          <h2 class="text-xs font-semibold uppercase tracking-wide text-ink-muted">{{ $t('pulls.discussion') }}</h2>
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
        </template>
        <template #comment-form>
          <form v-if="isAuthenticated" class="page-card space-y-2" @submit.prevent="addComment">
            <textarea v-model="comment" :class="['field', 'min-h-24']" required :placeholder="$t('pulls.comment')" />
            <button type="submit" class="btn-primary" :disabled="busy">{{ $t('pulls.commentAction') }}</button>
          </form>
        </template>
      </DiscussionDetailLayout>
    </PageState>
  </RepoShell>
</template>
