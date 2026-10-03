<script setup lang="ts">
import { ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { pullsApi, reposApi } from '@/api'
import type { PullRequest, RefInfo } from '@/api/types'
import RepoShell from '@/components/repo/RepoShell.vue'
import Icon from '@/components/ui/Icon.vue'
import PageState from '@/components/ui/PageState.vue'
import { formatDate, localizeError } from '@/i18n'

const props = defineProps<{
  owner: string
  repoPath: string
}>()

const state = ref<'open' | 'closed' | 'merged'>('open')
const pulls = ref<PullRequest[]>([])
const branches = ref<RefInfo[]>([])
const title = ref('')
const body = ref('')
const baseBranch = ref('main')
const headBranch = ref('')
const error = ref('')
const loading = ref(true)
const creating = ref(false)
const showForm = ref(false)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [pullRes, branchRes, repo] = await Promise.all([
      pullsApi.list(props.owner, props.repoPath, state.value),
      reposApi.branches(props.owner, props.repoPath),
      reposApi.get(props.owner, props.repoPath),
    ])
    pulls.value = pullRes.pulls ?? []
    branches.value = branchRes.branches ?? []
    baseBranch.value = repo.default_branch || 'main'
    if (!headBranch.value && branches.value.length) {
      const other = branches.value.find((b) => b.name !== baseBranch.value)
      headBranch.value = other?.name || branches.value[0]?.name || ''
    }
  } catch (err) {
    error.value = localizeError(err, 'errors.loadPulls')
  } finally {
    loading.value = false
  }
}

async function create() {
  creating.value = true
  error.value = ''
  try {
    await pullsApi.create(props.owner, props.repoPath, {
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

function iconClass(s: string) {
  if (s === 'open') {
    return 'text-open'
  }

  if (s === 'merged') {
    return 'text-done'
  }

  return 'text-ink-muted'
}

watch(() => [props.owner, props.repoPath, state.value], load, { immediate: true })
</script>

<template>
  <RepoShell 
    :owner="owner" 
    :repo-path="repoPath" 
    tab="pulls"
  >
    <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
      <div class="inline-flex overflow-hidden rounded-md border border-line bg-white text-sm">
        <button
          v-for="tab in (['open', 'closed', 'merged'] as const)"
          :key="tab"
          type="button"
          class="border-l border-line px-3 py-1.5 first:border-l-0"
          :class="state === tab ? 'bg-paper font-semibold' : 'text-ink-muted'"
          @click="state = tab"
        >
          {{ $t(`pulls.${tab}`) }}
        </button>
      </div>
      <button 
        type="button" 
        class="btn-primary" 
        @click="showForm = !showForm"
      >{{ $t('pulls.new') }}</button>
    </div>

    <form 
      v-if="showForm" 
      class="mb-4 space-y-3 rounded-md border border-line bg-white p-4" 
      @submit.prevent="create"
    >
      <h2 class="text-sm font-semibold">{{ $t('pulls.new') }}</h2>
      <input 
        v-model="title" 
        class="input" 
        required 
        :placeholder="$t('pulls.titleField')"
      />
      <textarea 
        v-model="body" 
        class="input min-h-24" 
        :placeholder="$t('pulls.bodyField')" 
      />
      <div class="grid gap-3 sm:grid-cols-2">
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('pulls.base') }}</span>
          <select v-model="baseBranch" class="input">
            <option 
              v-for="b in branches" 
              :key="'base-' + b.name" 
              :value="b.name"
            >{{ b.name }}</option>
          </select>
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('pulls.head') }}</span>
          <select v-model="headBranch" class="input" required>
            <option 
              v-for="b in branches" 
              :key="'head-' + b.name" 
              :value="b.name"
            >{{ b.name }}</option>
          </select>
        </label>
      </div>
      <button 
        type="submit" 
        class="btn-primary" 
        :disabled="creating"
      >
        {{ creating ? $t('pulls.creating') : $t('pulls.create') }}
      </button>
    </form>

    <p v-if="error" class="mb-3 text-sm text-warn">{{ error }}</p>
    <PageState :loading="loading">
      <ul class="divide-y divide-line overflow-hidden rounded-md border border-line bg-white">
        <li v-for="pull in pulls" :key="pull.number">
          <RouterLink
            :to="{ 
              name: 'repo-pull', 
              params: { 
                owner, repoPath, 
                number: String(pull.number) 
              } 
            }"
            class="flex items-start gap-3 px-4 py-3 hover:bg-paper"
          >
            <Icon :name="iconFor(pull.state)" class="mt-0.5" :class="iconClass(pull.state)" />
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
  </RepoShell>
</template>
