<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { issuesApi } from '@/api'
import type { IssueDetail, Label } from '@/api/types'
import CommentCard from '@/components/issues/CommentCard.vue'
import LabelPicker from '@/components/issues/LabelPicker.vue'
import RepoShell from '@/components/repo/RepoShell.vue'
import Icon from '@/components/ui/Icon.vue'
import LabelPill from '@/components/ui/LabelPill.vue'
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
const issue = ref<IssueDetail | null>(null)
const labels = ref<Label[]>([])
const comment = ref('')
const error = ref('')
const loading = ref(true)
const busy = ref(false)
const selected = ref<number[]>([])

const isOwner = computed(() => auth.user.value?.username === props.owner)
const canEdit = computed(() => isOwner.value || auth.user.value?.username === issue.value?.author)

function canEditComment(author: string) {
  return isOwner.value || auth.user.value?.username === author
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [detail, labelRes] = await Promise.all([
      issuesApi.get(props.owner, props.repoPath, Number(props.number)),
      issuesApi.labels(props.owner, props.repoPath),
    ])
    issue.value = detail
    labels.value = labelRes.labels ?? []
    selected.value = detail.labels.map((label) => label.id)
  } catch (err) {
    error.value = localizeError(err, 'errors.loadIssue')
  } finally {
    loading.value = false
  }
}

async function toggleState() {
  if (!issue.value) return
  busy.value = true
  error.value = ''
  try {
    const next = issue.value.state === 'open' ? 'closed' : 'open'
    await issuesApi.update(props.owner, props.repoPath, issue.value.number, { state: next })
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.saveFailed')
  } finally {
    busy.value = false
  }
}

async function saveLabels() {
  if (!issue.value) return
  busy.value = true
  error.value = ''
  try {
    await issuesApi.update(props.owner, props.repoPath, issue.value.number, { label_ids: selected.value })
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.saveFailed')
  } finally {
    busy.value = false
  }
}

async function addComment() {
  if (!issue.value) return
  busy.value = true
  error.value = ''
  try {
    await issuesApi.comment(props.owner, props.repoPath, issue.value.number, comment.value)
    comment.value = ''
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.createComment')
  } finally {
    busy.value = false
  }
}

async function saveComment(id: number, body: string) {
  if (!issue.value) return
  busy.value = true
  error.value = ''
  try {
    await issuesApi.updateComment(props.owner, props.repoPath, issue.value.number, id, body)
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.saveFailed')
  } finally {
    busy.value = false
  }
}

async function removeComment(id: number) {
  if (!issue.value) return
  busy.value = true
  error.value = ''
  try {
    await issuesApi.removeComment(props.owner, props.repoPath, issue.value.number, id)
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.deleteFailed')
  } finally {
    busy.value = false
  }
}

async function remove() {
  if (!issue.value || !confirm(`#${issue.value.number}`)) return
  busy.value = true
  try {
    await issuesApi.remove(props.owner, props.repoPath, issue.value.number)
    await router.push({
      name: 'repo-issues',
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
    tab="issues"
  >
    <PageState :loading="loading" :error="error">
      <template v-if="issue">
        <div class="mb-4 flex flex-wrap items-start justify-between gap-3">
          <h1 class="text-2xl font-semibold">
            {{ issue.title }}
            <span class="font-normal text-ink-muted">#{{ issue.number }}</span>
          </h1>
          <div class="flex gap-2">
            <button
              v-if="canEdit"
              type="button"
              class="btn-ghost"
              :disabled="busy"
              @click="toggleState"
            >{{ issue.state === 'open' ? $t('issues.close') : $t('issues.reopen') }}</button>
            <button
              v-if="canEdit"
              type="button"
              class="btn-danger"
              :disabled="busy"
              @click="remove"
            >{{ $t('issues.delete') }}</button>
          </div>
        </div>

        <div class="mb-4 flex items-center gap-2 text-sm">
          <span
            class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-semibold text-white"
            :class="issue.state === 'open' ? 'bg-open' : 'bg-done'"
          >
            <Icon :name="issue.state === 'open' ? 'issue' : 'issue-closed'" />
            {{ $t(`issues.${issue.state}`) }}
          </span>
          <span class="text-ink-muted">{{ issue.author }} {{ formatDate(issue.created_at) }}</span>
        </div>

        <div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_240px]">
          <div class="space-y-4">
            <article class="overflow-hidden rounded-md border border-line bg-white">
              <header class="border-b border-line bg-paper px-4 py-2 text-xs">
                <span class="font-semibold">{{ issue.author }}</span>
                <span class="text-ink-muted"> {{ formatDate(issue.created_at) }}</span>
              </header>
              <p class="whitespace-pre-wrap px-4 py-3 text-sm">{{ issue.body || $t('issues.noBody') }}</p>
            </article>

            <CommentCard
              v-for="item in issue.comments"
              :key="item.id"
              :author="item.author"
              :body="item.body"
              :created-at="item.created_at"
              :can-edit="canEditComment(item.author)"
              :busy="busy"
              @save="saveComment(item.id, $event)"
              @remove="removeComment(item.id)"
            />
            <p v-if="!issue.comments.length" class="text-sm text-ink-muted">{{ $t('issues.noComments') }}</p>

            <form class="space-y-2" @submit.prevent="addComment">
              <textarea
                v-model="comment"
                class="input min-h-24"
                required
                :placeholder="$t('issues.comment')"
              />
              <button
                type="submit"
                class="btn-primary"
                :disabled="busy"
              >{{ $t('issues.commentAction') }}</button>
            </form>
          </div>

          <aside class="space-y-4">
            <section class="rounded-md border border-line bg-white p-4">
              <h2 class="mb-2 text-xs font-semibold text-ink-muted">{{ $t('issues.labels') }}</h2>
              <div class="mb-2 flex flex-wrap gap-1">
                <LabelPill
                  v-for="label in issue.labels"
                  :key="label.id"
                  :label="label"
                />
                <span
                  v-if="!issue.labels.length"
                  class="text-xs text-ink-muted"
                >{{ $t('repo.noneYet') }}</span>
              </div>
              <form
                v-if="isOwner"
                class="space-y-2"
                @submit.prevent="saveLabels"
              >
                <LabelPicker
                v-model="selected"
                  :labels="labels"
                />
                <button
                  type="submit"
                  class="btn-ghost"
                  :disabled="busy"
                >{{ $t('issues.saveLabels') }}</button>
              </form>
            </section>
          </aside>
        </div>
      </template>
    </PageState>
  </RepoShell>
</template>
