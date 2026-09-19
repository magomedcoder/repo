<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { issuesApi } from '@/api'
import type { Issue, Label } from '@/api/types'
import IssueRow from '@/components/issues/IssueRow.vue'
import LabelManager from '@/components/issues/LabelManager.vue'
import LabelPicker from '@/components/issues/LabelPicker.vue'
import RepoShell from '@/components/repo/RepoShell.vue'
import Icon from '@/components/ui/Icon.vue'
import PageState from '@/components/ui/PageState.vue'
import { useAuth } from '@/composables/useAuth'
import { localizeError } from '@/i18n'

const props = defineProps<{
  owner: string
  repoPath: string
}>()

const auth = useAuth()
const state = ref<'open' | 'closed'>('open')
const issues = ref<Issue[]>([])
const labels = ref<Label[]>([])
const title = ref('')
const body = ref('')
const selected = ref<number[]>([])
const error = ref('')
const loading = ref(true)
const creating = ref(false)
const showForm = ref(false)

const isOwner = computed(() => auth.user.value?.username === props.owner)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [issueRes, labelRes] = await Promise.all([
      issuesApi.list(props.owner, props.repoPath, state.value),
      issuesApi.labels(props.owner, props.repoPath),
    ])
    issues.value = issueRes.issues ?? []
    labels.value = labelRes.labels ?? []
  } catch (err) {
    error.value = localizeError(err, 'errors.loadIssues')
  } finally {
    loading.value = false
  }
}

async function create() {
  creating.value = true
  error.value = ''
  try {
    await issuesApi.create(props.owner, props.repoPath, {
      title: title.value,
      body: body.value,
      label_ids: selected.value,
    })
    title.value = ''
    body.value = ''
    selected.value = []
    showForm.value = false
    state.value = 'open'
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.createIssue')
  } finally {
    creating.value = false
  }
}

async function addLabel(payload: { name: string; color: string }) {
  error.value = ''
  try {
    await issuesApi.createLabel(props.owner, props.repoPath, payload)
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.createLabel')
  }
}

async function removeLabel(id: number) {
  error.value = ''
  try {
    await issuesApi.removeLabel(props.owner, props.repoPath, id)
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.deleteFailed')
  }
}

watch(() => [props.owner, props.repoPath, state.value], load, { immediate: true })
</script>

<template>
  <RepoShell
    :owner="owner"
    :repo-path="repoPath"
    tab="issues"
  >
    <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
      <div class="inline-flex overflow-hidden rounded-md border border-line bg-white text-sm">
        <button
          type="button"
          class="inline-flex items-center gap-1.5 px-3 py-1.5"
          :class="state === 'open' ? 'bg-paper font-semibold' : 'text-ink-muted'"
          @click="state = 'open'"
        >
          <Icon name="issue" class="text-open" />
          {{ $t('issues.open') }}
        </button>
        <button
          type="button"
          class="inline-flex items-center gap-1.5 border-l border-line px-3 py-1.5"
          :class="state === 'closed' ? 'bg-paper font-semibold' : 'text-ink-muted'"
          @click="state = 'closed'"
        >
          <Icon name="issue-closed" class="text-done" />
          {{ $t('issues.closed') }}
        </button>
      </div>
      <button
        type="button"
        class="btn-primary"
        @click="showForm = !showForm"
      >{{ $t('issues.new') }}</button>
    </div>

    <form
      v-if="showForm"
      class="mb-4 space-y-3 rounded-md border border-line bg-white p-4"
      @submit.prevent="create"
    >
      <h2 class="text-sm font-semibold">{{ $t('issues.new') }}</h2>
      <input
        v-model="title"
        class="input"
        required
        :placeholder="$t('issues.titleField')"
      />
      <textarea
        v-model="body"
        class="input min-h-28"
        :placeholder="$t('issues.bodyField')"
      />
      <LabelPicker
        v-model="selected"
        :labels="labels"
      />
      <button
        type="submit"
        class="btn-primary"
        :disabled="creating"
      >
        {{ creating ? $t('issues.creating') : $t('issues.create') }}
      </button>
    </form>

    <p v-if="error" class="mb-3 text-sm text-warn">{{ error }}</p>
    <PageState :loading="loading">
      <div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_260px]">
        <ul class="divide-y divide-line overflow-hidden rounded-md border border-line bg-white">
          <li v-for="issue in issues" :key="issue.number">
            <IssueRow
              :issue="issue"
              :owner="owner"
              :repo-path="repoPath"
            />
          </li>
          <li v-if="!issues.length" class="px-4 py-10 text-center text-sm text-ink-muted">{{ $t('issues.empty') }}</li>
        </ul>
        <LabelManager
          v-if="isOwner"
          :labels="labels"
          @create="addLabel"
          @remove="removeLabel"
        />
      </div>
    </PageState>
  </RepoShell>
</template>
