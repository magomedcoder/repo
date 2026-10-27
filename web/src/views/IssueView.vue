<script setup lang="ts">
import RepoShell from '@/components/repo/RepoShell.vue'
import IssueBody from '@/views/repo/IssueBody.vue'

defineProps<{
  repoPath: string
  number: string
}>()
</script>

<template>
  <RepoShell :repo-path="repoPath" tab="issues">
    <IssueBody :repo-path="repoPath" :number="number" />
  </RepoShell>
</template>
