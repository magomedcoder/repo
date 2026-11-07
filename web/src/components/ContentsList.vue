<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import type { Repository, RepoSummary } from '@/api/types'
import EmptyState from '@/components/ui/EmptyState.vue'
import Icon from '@/components/ui/Icon.vue'
import { useAuth } from '@/composables/useAuth'
import { idleAge } from '@/lib/idle'
import { toContentRow } from '@/lib/contentRow'
import { repoLocation } from '@/lib/repoNav'

const props = defineProps<{
  repos: (RepoSummary | Repository)[]
  sort?: string
  emptyText?: string
  title?: string
  hint?: string
}>()

const auth = useAuth()

const rows = computed(() => props.repos.map((repo) => toContentRow(repo, auth.user.value?.username)))

const privacy = computed(() => {
  const map = new Map<number, boolean>()
  for (const repo of props.repos) {
    if ('is_private' in repo) {
      map.set(repo.id, !!repo.is_private)
    }
  }
  return map
})

const emit = defineEmits<{
  sort: [key: string]
}>()

const sortKeys = [
  { key: 'name', labelKey: 'index.name' },
  { key: 'idle', labelKey: 'index.idle' },
] as const

function repoTo(row: ReturnType<typeof toContentRow>) {
  return repoLocation('repo', row.repo_path)
}

function idleOf(row: ReturnType<typeof toContentRow>) {
  return idleAge(row.last_activity_at || row.updated_at)
}

function onSortChange(ev: Event) {
  const value = (ev.target as HTMLSelectElement).value
  emit('sort', value)
}

function isPrivate(id: number) {
  return privacy.value.get(id) === true
}
</script>

<template>
  <div class="glass overflow-hidden">
    <div
      v-if="title || sort"
      class="flex flex-wrap items-end justify-between gap-x-3 gap-y-1.5 border-b border-line-strong/40 px-3 py-2 sm:px-4"
    >
      <div v-if="title" class="min-w-0">
        <h2 class="text-[15px] font-semibold leading-snug text-ink">
          {{ title }}<template v-if="hint"><span class="font-normal text-ink-muted"> {{ hint }}</span></template>
        </h2>
      </div>
      <label
        v-if="sort !== undefined"
        class="inline-flex min-w-0 items-center gap-1.5 text-[12px] text-ink-muted"
      >
        <Icon name="sort" size="sm" class="shrink-0 opacity-70" />
        <span class="shrink-0 font-medium">{{ $t('home.sort') }}</span>
        <select
          class="field w-auto min-w-[7.5rem] py-1 text-[12px] text-ink"
          :value="sort"
          @change="onSortChange"
        >
          <option
            v-for="item in sortKeys"
            :key="item.key"
            :value="item.key"
          >
            {{ $t(item.labelKey) }}
          </option>
        </select>
      </label>
    </div>

    <EmptyState
      v-if="!rows.length"
      :message="emptyText || $t('list.noRepos')"
    />
    <ul
      v-else
      class="divide-y divide-line-strong/30"
    >
      <li
        v-for="row in rows"
        :key="`repo-${row.id}`"
      >
        <RouterLink
          :to="repoTo(row)"
          class="flex items-start gap-2 px-3 py-2 text-ink no-underline transition-colors hover:bg-row-hover sm:px-4"
        >
          <span class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full border border-line-strong/70 bg-white/40 text-ink-muted">
            <Icon name="repo" size="md" />
          </span>
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-x-1.5 gap-y-0.5">
              <h2 class="min-w-0 text-[14px] font-semibold leading-snug">
                <span class="text-link">{{ row.name }}</span>
              </h2>
              <span
                v-if="isPrivate(row.id)"
                class="inline-flex items-center rounded-full border border-line-strong/70 px-1.5 py-px text-[10px] font-medium leading-none text-ink-muted"
              >
                {{ $t('home.private') }}
              </span>
            </div>

            <p
              v-if="row.description"
              class="mt-0.5 line-clamp-2 text-[12px] leading-snug text-ink-muted"
            >
              {{ row.description }}
            </p>

            <p
              v-if="idleOf(row)"
              class="mt-1 text-[11px] text-ink-muted"
            >
              {{ $t('home.updated', { age: idleOf(row) }) }}
            </p>
          </div>
        </RouterLink>
      </li>
    </ul>
  </div>
</template>
