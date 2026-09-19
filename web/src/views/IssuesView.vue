<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { issuesApi } from '@/api'
import type { Issue, Label } from '@/api/types'
import { useAuth } from '@/composables/useAuth'
import { useBreadcrumbs, type Crumb } from '@/composables/useBreadcrumbs'
import { formatDate, localizeError } from '@/i18n'

const props = defineProps<{
  owner: string
  repoPath: string
}>()
const auth = useAuth()
const { setBreadcrumbs, clearBreadcrumbs } = useBreadcrumbs()

const state = ref<'open' | 'closed'>('open')
const issues = ref<Issue[]>([])
const labels = ref<Label[]>([])
const title = ref('')
const body = ref('')
const selected = ref<number[]>([])
const error = ref('')
const loading = ref(true)
const creating = ref(false)
const labelName = ref('')
const labelColor = ref('#1f6b4f')

const isOwner = computed(() => auth.user.value?.username === props.owner)

function crumbs() {
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
    labelKey: 'nav.issues'
  })
  setBreadcrumbs(items)
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    crumbs()
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
    state.value = 'open'
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.createIssue')
  } finally {
    creating.value = false
  }
}

async function addLabel() {
  error.value = ''
  try {
    await issuesApi.createLabel(props.owner, props.repoPath, {
      name: labelName.value,
      color: labelColor.value
    })
    labelName.value = ''
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
onUnmounted(clearBreadcrumbs)
</script>

<template>
  <div>
    <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
      <h1 class="font-display text-2xl font-bold">{{ $t('issues.title') }}</h1>
      <RouterLink
        class="btn-ghost"
        :to="{
          name: 'repo',
          params: { owner, repoPath }
        }"
      >{{ $t('repo.home') }}</RouterLink>
    </div>

    <div class="mb-4 flex gap-2">
      <button
        type="button"
        class="btn-ghost"
        :class="state === 'open' ? 'bg-moss-soft' : ''"
        @click="state = 'open'"
      >{{ $t('issues.open') }}</button>
      <button
        type="button"
        class="btn-ghost"
        :class="state === 'closed' ? 'bg-moss-soft' : ''"
        @click="state = 'closed'"
      >{{ $t('issues.closed') }}</button>
    </div>

    <p v-if="loading" class="text-sm text-ink-muted">{{ $t('common.loading') }}</p>
    <p v-else-if="error" class="mb-3 text-sm text-warn">{{ error }}</p>

    <ul v-if="!loading" class="mb-6 divide-y divide-line overflow-hidden rounded-lg border border-line bg-white/80">
      <li v-for="issue in issues" :key="issue.number">
        <RouterLink
          :to="{
            name: 'repo-issue',
            params: { owner, repoPath, number: String(issue.number) }
          }"
          class="block px-4 py-3 hover:bg-moss-soft/40"
        >
          <div class="flex flex-wrap items-center gap-2">
            <p class="font-semibold">{{ issue.title }}</p>
            <span
              v-for="label in issue.labels"
              :key="label.id"
              class="rounded px-1.5 py-0.5 text-[10px] font-semibold text-white"
              :style="{ background: label.color }"
            >{{ label.name }}</span>
          </div>
          <p class="mt-1 font-mono text-xs text-ink-muted">#{{ issue.number }} {{ issue.author }} {{ formatDate(issue.created_at) }} {{ issue.comment_count }}</p>
        </RouterLink>
      </li>
      <li v-if="!issues.length" class="px-4 py-8 text-center text-sm text-ink-muted">{{ $t('issues.empty') }}</li>
    </ul>

    <form class="panel mb-6 space-y-3 p-4" @submit.prevent="create">
      <h2 class="font-display text-lg font-bold">{{ $t('issues.new') }}</h2>
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
      <div v-if="labels.length" class="flex flex-wrap gap-2">
        <label 
          v-for="label in labels" 
          :key="label.id" 
          class="flex items-center gap-1 text-sm"
        >
          <input
            v-model="selected"
            type="checkbox"
            :value="label.id"
            class="accent-moss"
          />
          <span
            class="rounded px-1.5 py-0.5 text-xs text-white"
            :style="{ background: label.color }"
          >{{ label.name }}</span>
        </label>
      </div>
      <button
        type="submit"
        class="btn-primary"
        :disabled="creating"
      >{{ creating ? $t('issues.creating') : $t('issues.create') }}</button>
    </form>

    <section v-if="isOwner" class="panel space-y-3 p-4">
      <h2 class="font-display text-lg font-bold">{{ $t('issues.labels') }}</h2>
      <ul class="space-y-1">
        <li
          v-for="label in labels"
          :key="label.id"
          class="flex items-center justify-between gap-2"
        >
          <span
            class="rounded px-2 py-0.5 text-xs text-white"
            :style="{ background: label.color }"
          >{{ label.name }}</span>
          <button
            type="button"
            class="text-xs text-warn"
            @click="removeLabel(label.id)"
          >{{ $t('issues.delete') }}</button>
        </li>
      </ul>
      <form
        class="flex flex-wrap gap-2"
        @submit.prevent="addLabel"
      >
        <input
          v-model="labelName"
          class="input max-w-48"
          required
          :placeholder="$t('issues.labelName')"
        />
        <input
          v-model="labelColor"
          type="color"
          class="h-10 w-12 rounded border border-line bg-white"
        />
        <button
          type="submit"
          class="btn-ghost"
        >{{ $t('issues.addLabel') }}</button>
      </form>
    </section>
  </div>
</template>
