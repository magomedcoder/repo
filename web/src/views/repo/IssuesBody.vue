<script setup lang="ts">
import { btnPrimary, field } from '@/lib/classes'
import { computed, ref, watch } from 'vue'
import { issuesApi } from '@/api'
import type { Issue, Label } from '@/api/types'
import IssueRow from '@/components/issues/IssueRow.vue'
import LabelManager from '@/components/issues/LabelManager.vue'
import LabelPicker from '@/components/issues/LabelPicker.vue'
import Icon from '@/components/ui/Icon.vue'
import PageState from '@/components/ui/PageState.vue'
import { repoPathMatches, useInjectedRepo } from '@/composables/useInjectedRepo'
import { useAuth } from '@/composables/useAuth'
import { localizeError } from '@/i18n'

const props = defineProps<{
  repoPath: string
}>()

const auth = useAuth()
const repo = useInjectedRepo()
const state = ref<'open' | 'closed'>('open')
const issues = ref<Issue[]>([])
const labels = ref<Label[]>([])
const title = ref('')
const body = ref('')
const selected = ref<number[]>([])
const error = ref('')
const dataLoading = ref(false)
const creating = ref(false)
const showForm = ref(false)

const shellLoading = computed(() => !repo.value || !repoPathMatches(repo.value, props.repoPath))
const loading = computed(() => shellLoading.value || dataLoading.value)
const isOwner = computed(
  () => !!auth.user.value?.username && !!repo.value && auth.user.value.username === repo.value.owner,
)

async function loadIssues() {
  if (!repo.value || !repoPathMatches(repo.value, props.repoPath)) {
    return
  }
  dataLoading.value = true
  error.value = ''
  try {
    const [issueRes, labelRes] = await Promise.all([
      issuesApi.list(props.repoPath, state.value),
      issuesApi.labels(props.repoPath),
    ])
    issues.value = issueRes.issues ?? []
    labels.value = labelRes.labels ?? []
  } catch (err) {
    error.value = localizeError(err, 'errors.loadIssues')
  } finally {
    dataLoading.value = false
  }
}

async function create() {
  creating.value = true
  error.value = ''
  try {
    await issuesApi.create(props.repoPath, {
      title: title.value,
      body: body.value,
      label_ids: selected.value,
    })
    title.value = ''
    body.value = ''
    selected.value = []
    showForm.value = false
    state.value = 'open'
    await loadIssues()
  } catch (err) {
    error.value = localizeError(err, 'errors.createIssue')
  } finally {
    creating.value = false
  }
}

async function addLabel(payload: { name: string; color: string }) {
  error.value = ''
  try {
    await issuesApi.createLabel(props.repoPath, payload)
    await loadIssues()
  } catch (err) {
    error.value = localizeError(err, 'errors.createLabel')
  }
}

async function removeLabel(id: number) {
  error.value = ''
  try {
    await issuesApi.removeLabel(props.repoPath, id)
    await loadIssues()
  } catch (err) {
    error.value = localizeError(err, 'errors.deleteFailed')
  }
}

watch([() => props.repoPath, () => state.value, repo], loadIssues, { immediate: true })
</script>

<template>
  <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
    <div class="inline-flex overflow-hidden border border-line bg-paper-2 text-sm">
      <button
        type="button"
        class="inline-flex items-center gap-1.5 px-3 py-1.5"
        :class="state === 'open' ? 'bg-paper font-medium text-ink' : 'text-ink-muted hover:text-ink'"
        @click="state = 'open'"
      >
        <Icon name="issue" class="text-ink-muted" />
        {{ $t('issues.open') }}
      </button>
      <button
        type="button"
        class="inline-flex items-center gap-1.5 border-l border-line px-3 py-1.5"
        :class="state === 'closed' ? 'bg-paper font-medium text-ink' : 'text-ink-muted hover:text-ink'"
        @click="state = 'closed'"
      >
        <Icon name="issue-closed" class="text-ink-muted" />
        {{ $t('issues.closed') }}
      </button>
    </div>
    <button type="button" :class="btnPrimary" @click="showForm = !showForm">{{ $t('issues.new') }}</button>
  </div>

  <form v-if="showForm" class="term-block mb-4" @submit.prevent="create">
    <div class="term-block-hd">{{ $t('issues.new') }}</div>
    <div class="term-block-bd space-y-3">
      <input v-model="title" :class="field" required :placeholder="$t('issues.titleField')" />
      <textarea v-model="body" :class="[field, 'min-h-28']" :placeholder="$t('issues.bodyField')" />
      <LabelPicker v-model="selected" :labels="labels" />
      <button type="submit" :class="btnPrimary" :disabled="creating">
        {{ creating ? $t('issues.creating') : $t('issues.create') }}
      </button>
    </div>
  </form>

  <p v-if="error" class="mb-3 text-sm text-warn">{{ error }}</p>
  <PageState :loading="loading">
    <div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_260px]">
      <ul class="term-block divide-y divide-line">
        <li v-for="issue in issues" :key="issue.number">
          <IssueRow :issue="issue" :repo-path="repoPath" />
        </li>
        <li v-if="!issues.length" class="px-4 py-10 text-center text-sm text-ink-muted">{{ $t('issues.empty') }}</li>
      </ul>
      <LabelManager v-if="isOwner" :labels="labels" @create="addLabel" @remove="removeLabel" />
    </div>
  </PageState>
</template>
