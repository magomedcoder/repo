<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { issuesApi } from '@/api'
import type { Issue, Label } from '@/api/types'
import IssueRow from '@/components/issues/IssueRow.vue'
import LabelManager from '@/components/issues/LabelManager.vue'
import LabelPicker from '@/components/issues/LabelPicker.vue'
import RepoShell from '@/components/repo/RepoShell.vue'
import Icon from '@/components/ui/Icon.vue'
import PageState from '@/components/ui/PageState.vue'
import ActionBar from '@/components/ui/ActionBar.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SegmentedControl from '@/components/ui/SegmentedControl.vue'
import Surface from '@/components/ui/Surface.vue'
import SurfaceHeader from '@/components/ui/SurfaceHeader.vue'
import { useCanWrite } from '@/composables/useCanWrite'
import { useRepo } from '@/composables/useRepo'
import { i18n, localizeError } from '@/i18n'
import { repoLocation } from '@/lib/repoNav'

const props = defineProps<{
  repoPath: string
}>()

const router = useRouter()
const { repo, loading: repoLoading, error: repoError } = useRepo(() => props.repoPath)

watch(repo, (value) => {
  if (value && value.issues_enabled === false) {
    void router.replace(repoLocation('repo', props.repoPath))
  }
})
const { isAuthenticated, canAdmin } = useCanWrite(repo)
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

const loading = computed(() => dataLoading.value)
const stateOptions = computed(() => [
  { 
    value: 'open' as const, 
    label: i18n.global.t('issues.open'), 
    icon: 'issue' 
  },
  { 
    value: 'closed' as const, 
    label: i18n.global.t('issues.closed'), 
    icon: 'issue-closed' 
  },
])

async function loadIssues() {
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

watch(isAuthenticated, (ok) => {
  if (!ok) {
    showForm.value = false
  }
})

watch([() => props.repoPath, () => state.value], loadIssues, { immediate: true })
</script>

<template>
  <RepoShell
    :repo-path="repoPath"
    tab="issues"
    :repo="repo"
    :loading="repoLoading"
    :error="repoError"
  >
    <ActionBar>
      <template #leading>
        <SegmentedControl v-model="state" :options="stateOptions">
          <template #option="{ option }">
            <Icon :name="option.icon as 'issue' | 'issue-closed'" class="text-ink-muted" />
            {{ option.label }}
          </template>
        </SegmentedControl>
      </template>
      <template #actions>
        <button
          v-if="isAuthenticated"
          type="button"
          class="btn-primary"
          @click="showForm = !showForm"
        ><Icon name="plus" size="sm" />{{ $t('issues.new') }}</button>
      </template>
    </ActionBar>

    <Surface
      v-if="isAuthenticated && showForm"
      as="form"
      class="mb-4"
      @submit.prevent="create"
    >
      <SurfaceHeader>{{ $t('issues.new') }}</SurfaceHeader>
      <div class="p-3 sm:p-4 space-y-3">
        <input v-model="title" class="field" required :placeholder="$t('issues.titleField')" />
        <textarea v-model="body" :class="['field', 'min-h-28']" :placeholder="$t('issues.bodyField')" />
        <LabelPicker v-model="selected" :labels="labels" />
        <button type="submit" class="btn-primary" :disabled="creating">
          {{ creating ? $t('issues.creating') : $t('issues.create') }}
        </button>
      </div>
    </Surface>

    <p v-if="error" class="mb-3 text-sm text-warn">{{ error }}</p>
    <PageState :loading="loading">
      <div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_260px]">
        <Surface as="ul" class="divide-y divide-line">
          <li v-for="issue in issues" :key="issue.number">
            <IssueRow :issue="issue" :repo-path="repoPath" />
          </li>
          <EmptyState
            v-if="!issues.length"
            as="li"
            :message="$t('issues.empty')"
          />
        </Surface>
        <LabelManager
          v-if="canAdmin"
          :labels="labels"
          @create="addLabel"
          @remove="removeLabel"
        />
      </div>
    </PageState>
  </RepoShell>
</template>
