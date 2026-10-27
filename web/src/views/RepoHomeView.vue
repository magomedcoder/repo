<script setup lang="ts">
import RepoShell from '@/components/repo/RepoShell.vue'
import RepoHomeBody from '@/views/repo/RepoHomeBody.vue'

defineProps<{
  repoPath: string
}>()
</script>

<template>
  <RepoShell :repo-path="repoPath" tab="code">
    <RepoHomeBody :repo-path="repoPath" />
  </RepoShell>
</template>
