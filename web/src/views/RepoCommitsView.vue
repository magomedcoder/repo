<script setup lang="ts">
import RepoShell from '@/components/repo/RepoShell.vue'
import RepoCommitsBody from '@/views/repo/RepoCommitsBody.vue'

defineProps<{
  repoPath: string
}>()
</script>

<template>
  <RepoShell :repo-path="repoPath" tab="code">
    <RepoCommitsBody :repo-path="repoPath" />
  </RepoShell>
</template>
