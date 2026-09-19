<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { issuesApi } from '@/api'
import type { IssueDetail, Label } from '@/api/types'
import { useAuth } from '@/composables/useAuth'
import { useBreadcrumbs, type Crumb } from '@/composables/useBreadcrumbs'
import { formatDate, localizeError } from '@/i18n'

const props = defineProps<{
  owner: string
  repoPath: string
  number: string
}>()
const auth = useAuth()
const router = useRouter()
const { setBreadcrumbs, clearBreadcrumbs } = useBreadcrumbs()

const issue = ref<IssueDetail | null>(null)
const labels = ref<Label[]>([])
const comment = ref('')
const editingId = ref<number | null>(null)
const editBody = ref('')
const error = ref('')
const loading = ref(true)
const busy = ref(false)
const selected = ref<number[]>([])

const isOwner = computed(() => auth.user.value?.username === props.owner)
const canEdit = computed(() => isOwner.value || auth.user.value?.username === issue.value?.author)

function setCrumbs() {
  const items: Crumb[] = [{
    label: props.owner,
    to: '/'
  }]
  props.repoPath.split('/').forEach((part, i, arr) => {
    items.push(i === arr.length - 1
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
  items.push({
    label: 'issues',
    labelKey: 'nav.issues',
    to: {
      name: 'repo-issues',
      params: {
        owner: props.owner,
        repoPath: props.repoPath
      }
    },
  })
  items.push({ label: `#${props.number}` })
  setBreadcrumbs(items)
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    setCrumbs()
    const [detail, labelRes] = await Promise.all([
      issuesApi.get(props.owner, props.repoPath, Number(props.number)),
      issuesApi.labels(props.owner, props.repoPath),
    ])
    issue.value = detail
    labels.value = labelRes.labels ?? []
    selected.value = detail.labels.map((l) => l.id)
  } catch (err) {
    error.value = localizeError(err, 'errors.loadIssue')
  } finally {
    loading.value = false
  }
}

async function toggleState() {
  if (!issue.value) {
    return
  }
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
  if (!issue.value) {
    return
  }
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
  if (!issue.value) {
    return
  }
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

async function saveComment(id: number) {
  if (!issue.value) {
    return
  }
  busy.value = true
  error.value = ''
  try {
    await issuesApi.updateComment(props.owner, props.repoPath, issue.value.number, id, editBody.value)
    editingId.value = null
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.saveFailed')
  } finally {
    busy.value = false
  }
}

async function removeComment(id: number) {
  if (!issue.value) {
    return
  }
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

function canEditComment(author: string) {
  return isOwner.value || auth.user.value?.username === author
}

async function remove() {
  if (!issue.value || !confirm(`#${issue.value.number}`)) {
    return
  }
  busy.value = true
  try {
    await issuesApi.remove(props.owner, props.repoPath, issue.value.number)
    await router.push({ name: 'repo-issues', params: { owner: props.owner, repoPath: props.repoPath } })
  } catch (err) {
    error.value = localizeError(err, 'errors.deleteFailed')
    busy.value = false
  }
}

watch(() => [props.owner, props.repoPath, props.number], load, { immediate: true })
onUnmounted(clearBreadcrumbs)
</script>

<template>
  <div>
    <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
      <RouterLink
        class="btn-ghost"
        :to="{
          name: 'repo-issues',
          params: { owner, repoPath }
        }"
      >{{ $t('issues.back') }}</RouterLink>
    </div>

    <p v-if="loading" class="text-sm text-ink-muted">{{ $t('common.loading') }}</p>
    <p v-else-if="error" class="text-sm text-warn">{{ error }}</p>
    <template v-else-if="issue">
      <div class="mb-4 flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 class="font-display text-3xl font-bold">{{ issue.title }} <span class="text-ink-muted">#{{ issue.number }}</span></h1>
          <p class="mt-1 font-mono text-xs text-ink-muted">{{ issue.author }} {{ formatDate(issue.created_at) }} {{ $t(`issues.${issue.state}`) }}</p>
        </div>
        <div class="flex gap-2">
          <button
            v-if="canEdit"
            type="button"
            class="btn-ghost"
            :disabled="busy"
            @click="toggleState"
          >
            {{ issue.state === 'open' ? $t('issues.close') : $t('issues.reopen') }}
          </button>
          <button
            v-if="canEdit"
            type="button"
            class="btn-danger"
            :disabled="busy"
            @click="remove"
          >{{ $t('issues.delete') }}</button>
        </div>
      </div>

      <div class="mb-4 flex flex-wrap gap-2">
        <span
          v-for="label in issue.labels"
          :key="label.id"
          class="rounded px-2 py-0.5 text-xs text-white"
          :style="{
            background: label.color
          }"
        >{{ label.name }}</span>
      </div>

      <article class="panel mb-6 whitespace-pre-wrap p-4 text-sm">{{ issue.body || $t('issues.noBody') }}</article>

      <form
        v-if="isOwner"
        class="panel mb-6 space-y-2 p-4"
        @submit.prevent="saveLabels"
      >
        <h2 class="text-sm font-semibold">{{ $t('issues.labels') }}</h2>
        <label
          v-for="label in labels"
          :key="label.id"
          class="mr-3 inline-flex items-center gap-1 text-sm"
        >
          <input
            v-model="selected"
            type="checkbox"
            :value="label.id"
            class="accent-moss"
          />
          {{ label.name }}
        </label>
        <div>
          <button
            type="submit"
            class="btn-ghost"
            :disabled="busy"
          >{{ $t('issues.saveLabels') }}</button>
        </div>
      </form>

      <h2 class="mb-2 text-xs font-semibold uppercase tracking-wide text-ink-muted">{{ $t('issues.discussion') }}</h2>
      <ul class="mb-4 space-y-3">
        <li
          v-for="item in issue.comments"
          :key="item.id"
          class="panel p-4"
        >
          <div class="flex items-center justify-between gap-2">
            <p class="font-mono text-xs text-ink-muted">{{ item.author }} {{ formatDate(item.created_at) }}</p>
            <div v-if="canEditComment(item.author)" class="flex gap-2">
              <button
                type="button"
                class="text-xs text-moss"
                @click="editingId = item.id; editBody = item.body"
              >{{ $t('issues.editComment') }}</button>
              <button
                type="button"
                class="text-xs text-warn"
                :disabled="busy"
                @click="removeComment(item.id)"
              >{{ $t('issues.delete') }}</button>
            </div>
          </div>
          <form
            v-if="editingId === item.id"
            class="mt-2 space-y-2"
            @submit.prevent="saveComment(item.id)"
          >
            <textarea
              v-model="editBody"
              class="input min-h-20"
              required
            />
            <button
              type="submit"
              class="btn-ghost"
              :disabled="busy"
            >{{ $t('issues.saveComment') }}</button>
          </form>
          <p v-else class="mt-2 whitespace-pre-wrap text-sm">{{ item.body }}</p>
        </li>
        <li v-if="!issue.comments.length" class="text-sm text-ink-muted">{{ $t('issues.noComments') }}</li>
      </ul>

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
    </template>
  </div>
</template>
