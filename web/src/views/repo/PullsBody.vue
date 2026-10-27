<script setup lang="ts">
import { btnPrimary, field } from '@/lib/classes'
import { computed, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { pullsApi, reposApi } from '@/api'
import type { PullRequest, RefInfo } from '@/api/types'
import Icon from '@/components/ui/Icon.vue'
import PageState from '@/components/ui/PageState.vue'
import { repoPathMatches, useInjectedRepo } from '@/composables/useInjectedRepo'
import { formatDate, localizeError } from '@/i18n'

const props = defineProps<{
  repoPath: string
}>()

const repo = useInjectedRepo()
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

const loading = computed(
  () => !repo.value || !repoPathMatches(repo.value, props.repoPath) || dataLoading.value,
)

async function load() {
  if (!repo.value || !repoPathMatches(repo.value, props.repoPath)) {
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

watch([() => props.repoPath, () => state.value, repo], load, { immediate: true })
</script>

<template>
  <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
    <div class="inline-flex overflow-hidden border border-line bg-paper-2 text-sm">
      <button
        v-for="tab in (['open', 'closed', 'merged'] as const)"
        :key="tab"
        type="button"
        class="border-l border-line px-3 py-1.5 first:border-l-0"
        :class="state === tab ? 'bg-paper font-medium text-ink' : 'text-ink-muted hover:text-ink'"
        @click="state = tab"
      >
        {{ $t(`pulls.${tab}`) }}
      </button>
    </div>
    <button type="button" :class="btnPrimary" @click="showForm = !showForm">{{ $t('pulls.new') }}</button>
  </div>

  <form v-if="showForm" class="term-block mb-4" @submit.prevent="create">
    <div class="term-block-hd">{{ $t('pulls.new') }}</div>
    <div class="term-block-bd space-y-3">
      <input v-model="title" :class="field" required :placeholder="$t('pulls.titleField')" />
      <textarea v-model="body" :class="[field, 'min-h-24']" :placeholder="$t('pulls.bodyField')" />
      <div class="grid gap-3 sm:grid-cols-2">
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('pulls.base') }}</span>
          <select v-model="baseBranch" :class="field">
            <option v-for="b in branches" :key="'base-' + b.name" :value="b.name">{{ b.name }}</option>
          </select>
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('pulls.head') }}</span>
          <select v-model="headBranch" :class="field" required>
            <option v-for="b in branches" :key="'head-' + b.name" :value="b.name">{{ b.name }}</option>
          </select>
        </label>
      </div>
      <button type="submit" :class="btnPrimary" :disabled="creating">
        {{ creating ? $t('pulls.creating') : $t('pulls.create') }}
      </button>
    </div>
  </form>

  <p v-if="error" class="mb-3 text-sm text-warn">{{ error }}</p>
  <PageState :loading="loading">
    <ul class="term-block divide-y divide-line">
      <li v-for="pull in pulls" :key="pull.number">
        <RouterLink
          :to="{ name: 'repo-pull', params: { repoPath, number: String(pull.number) } }"
          class="flex items-start gap-3 px-4 py-3 hover:bg-paper"
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
      <li v-if="!pulls.length" class="px-4 py-10 text-center text-sm text-ink-muted">{{ $t('pulls.empty') }}</li>
    </ul>
  </PageState>
</template>
