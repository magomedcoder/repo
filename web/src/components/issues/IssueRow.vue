<script setup lang="ts">
import { RouterLink } from 'vue-router'
import type { Issue } from '@/api/types'
import Icon from '@/components/ui/Icon.vue'
import LabelPill from '@/components/ui/LabelPill.vue'
import { formatDate } from '@/i18n'
import { repoLocation } from '@/lib/repoNav'

defineProps<{
  issue: Issue
  repoPath: string
}>()
</script>

<template>
  <RouterLink
    :to="repoLocation('repo-issue', repoPath, { number: issue.number })"
    class="flex items-start gap-3 px-4 py-3 hover:bg-row-hover"
  >
    <Icon
      :name="issue.state === 'open' ? 'issue' : 'issue-closed'"
      class="mt-0.5 text-ink-muted"
    />
    <div class="min-w-0 flex-1">
      <div class="flex flex-wrap items-center gap-2">
        <span class="font-semibold text-ink">{{ issue.title }}</span>
        <LabelPill
          v-for="label in issue.labels"
          :key="label.id"
          :label="label"
        />
      </div>
      <p class="mt-0.5 text-xs text-ink-muted">#{{ issue.number }} {{ issue.author }} {{ formatDate(issue.created_at) }}</p>
    </div>
    <span v-if="issue.comment_count" class="text-xs text-ink-muted">{{ issue.comment_count }}</span>
  </RouterLink>
</template>
