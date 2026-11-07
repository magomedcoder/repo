<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { pullsApi, reposApi } from '@/api'
import type { PullRequest, RefInfo } from '@/api/types'
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
import { formatDate, i18n, localizeError } from '@/i18n'
import { repoLocation } from '@/lib/repoNav'

const props = defineProps<{
  repoPath: string
}>()

const { repo, loading: repoLoading, error: repoError } = useRepo(() => props.repoPath)
const { isAuthenticated } = useCanWrite(repo)
const state = ref<'open' | 'closed' | 'merged'>('open')
const pulls = ref<PullRequest[]>([])
const branches = ref<RefInfo[]>([])
const title = ref('')
const body = ref('')
const baseBranch = ref('main')
const headBranch = ref('')
const error = ref('')
const dataLoading = ref(false)
const creating = ref(false)
const showForm = ref(false)

const loading = computed(() => dataLoading.value)
const stateOptions = computed(() => [
  { 
    value: 'open' as const, 
    label: i18n.global.t('pulls.open') 
  },
  { 
    value: 'closed' as const, 
    label: i18n.global.t('pulls.closed') 
  },
  { 
    value: 'merged' as const, 
    label: i18n.global.t('pulls.merged') 
  },
])

async function load() {
  if (!repo.value) {
    return
  }

  dataLoading.value = true
  error.value = ''
  try {
    const [pullRes, branchRes] = await Promise.all([
      pullsApi.list(props.repoPath, state.value),
      reposApi.branches(props.repoPath),
    ])
    pulls.value = pullRes.pulls ?? []
    branches.value = branchRes.branches ?? []
    baseBranch.value = repo.value.default_branch || 'main'
    if (!headBranch.value && branches.value.length) {
      const other = branches.value.find((b) => b.name !== baseBranch.value)
      headBranch.value = other?.name || branches.value[0]?.name || ''
    }
  } catch (err) {
    error.value = localizeError(err, 'errors.loadPulls')
  } finally {
    dataLoading.value = false
  }
}

async function create() {
  creating.value = true
  error.value = ''
  try {
    await pullsApi.create(props.repoPath, {
      title: title.value,
      body: body.value,
      base_branch: baseBranch.value,
      head_branch: headBranch.value,
    })
    title.value = ''
    body.value = ''
    showForm.value = false
    state.value = 'open'
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.createPull')
  } finally {
    creating.value = false
  }
}

function iconFor(s: string) {
  return s === 'merged' ? 'pull-merged' : 'pull'
}

watch(isAuthenticated, (ok) => {
  if (!ok) {
    showForm.value = false
  }
})

watch([() => props.repoPath, () => state.value, repo], load, { immediate: true })
</script>

<template>
  <RepoShell
    :repo-path="repoPath"
    tab="pulls"
    :repo="repo"
    :loading="repoLoading"
    :error="repoError"
  >
    <ActionBar>
      <template #leading>
        <SegmentedControl v-model="state" :options="stateOptions" />
      </template>
      <template #actions>
        <button
          v-if="isAuthenticated"
          type="button"
          class="btn-primary"
          @click="showForm = !showForm"
        ><Icon name="plus" size="sm" />{{ $t('pulls.new') }}</button>
      </template>
    </ActionBar>

    <Surface
      v-if="isAuthenticated && showForm"
      as="form"
      class="mb-4"
      @submit.prevent="create"
    >
      <SurfaceHeader>{{ $t('pulls.new') }}</SurfaceHeader>
      <div class="p-3 sm:p-4 space-y-3">
        <input v-model="title" class="field" required :placeholder="$t('pulls.titleField')" />
        <textarea v-model="body" :class="['field', 'min-h-24']" :placeholder="$t('pulls.bodyField')" />
        <div class="grid gap-3 sm:grid-cols-2">
          <label class="block text-sm">
            <span class="mb-1 block font-semibold">{{ $t('pulls.base') }}</span>
            <select v-model="baseBranch" class="field">
              <option v-for="b in branches" :key="'base-' + b.name" :value="b.name">{{ b.name }}</option>
            </select>
          </label>
          <label class="block text-sm">
            <span class="mb-1 block font-semibold">{{ $t('pulls.head') }}</span>
            <select v-model="headBranch" class="field" required>
              <option v-for="b in branches" :key="'head-' + b.name" :value="b.name">{{ b.name }}</option>
            </select>
          </label>
        </div>
        <button type="submit" class="btn-primary" :disabled="creating">
          {{ creating ? $t('pulls.creating') : $t('pulls.create') }}
        </button>
      </div>
    </Surface>

    <p v-if="error" class="mb-3 text-sm text-warn">{{ error }}</p>
    <PageState :loading="loading">
      <Surface as="ul" class="divide-y divide-line">
        <li v-for="pull in pulls" :key="pull.number">
          <RouterLink
            :to="repoLocation('repo-pull', repoPath, { number: pull.number })"
            class="flex items-start gap-3 px-4 py-3 hover:bg-row-hover"
          >
            <Icon :name="iconFor(pull.state)" class="mt-0.5 text-ink-muted" />
            <div class="min-w-0 flex-1">
              <p class="font-semibold text-ink">{{ pull.title }}</p>
              <p class="mt-0.5 text-xs text-ink-muted">
                #{{ pull.number }} {{ pull.author }} {{ pull.head_branch }} -> {{ pull.base_branch }} {{ formatDate(pull.created_at) }}
              </p>
            </div>
            <span v-if="pull.comment_count" class="text-xs text-ink-muted">{{ pull.comment_count }}</span>
          </RouterLink>
        </li>
        <EmptyState
          v-if="!pulls.length"
          as="li"
          :message="$t('pulls.empty')"
        />
      </Surface>
    </PageState>
  </RepoShell>
</template>
