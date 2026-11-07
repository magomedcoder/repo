<script setup lang="ts">
import { RouterLink } from 'vue-router'
import Icon from '@/components/ui/Icon.vue'
import { repoLocation } from '@/lib/repoNav'

const props = defineProps<{
  repoPath: string
  tab: 'summary' | 'log' | 'commit' | 'refs' | 'issues' | 'pulls' | 'settings'
  issuesEnabled?: boolean
  canAdmin?: boolean
}>()

function active(name: string) {
  return props.tab === name
}

function tabClass(name: string) {
  return [
    'repo-tab',
    active(name) && 'repo-tab-active',
  ]
}
</script>

<template>
  <nav class="repo-tabs" :aria-label="$t('nav.repo')">
    <RouterLink :to="repoLocation('repo', repoPath)" :class="tabClass('summary')">
      <Icon name="book" size="sm" />
      {{ $t('nav.code') }}
    </RouterLink>
    <RouterLink :to="repoLocation('repo-commits', repoPath)" :class="tabClass('log')">
      <Icon name="commit" size="sm" />
      {{ $t('nav.commits') }}
    </RouterLink>
    <RouterLink :to="repoLocation('repo-refs', repoPath)" :class="tabClass('refs')">
      <Icon name="branch" size="sm" />
      {{ $t('nav.refs') }}
    </RouterLink>
    <RouterLink
      v-if="issuesEnabled !== false"
      :to="repoLocation('repo-issues', repoPath)"
      :class="tabClass('issues')"
    >
      <Icon name="issue" size="sm" />
      {{ $t('nav.issues') }}
    </RouterLink>
    <RouterLink :to="repoLocation('repo-pulls', repoPath)" :class="tabClass('pulls')">
      <Icon name="pull" size="sm" />
      {{ $t('nav.pulls') }}
    </RouterLink>
    <RouterLink
      v-if="canAdmin"
      :to="repoLocation('repo-settings', repoPath)"
      :class="tabClass('settings')"
    >
      <Icon name="gear" size="sm" />
      {{ $t('nav.settings') }}
    </RouterLink>
  </nav>
</template>
