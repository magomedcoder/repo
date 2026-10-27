<script setup lang="ts">
import RepoShell from '@/components/repo/RepoShell.vue'
import IssuesBody from '@/views/repo/IssuesBody.vue'

defineProps<{
  repoPath: string
}>()
</script>

<template>
  <RepoShell :repo-path="repoPath" tab="issues">
    <IssuesBody :repo-path="repoPath" />
  </RepoShell>
</template>
