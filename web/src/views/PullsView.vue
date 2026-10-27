<script setup lang="ts">
import RepoShell from '@/components/repo/RepoShell.vue'
import PullsBody from '@/views/repo/PullsBody.vue'

defineProps<{
  repoPath: string
}>()
</script>

<template>
  <RepoShell :repo-path="repoPath" tab="pulls">
    <PullsBody :repo-path="repoPath" />
  </RepoShell>
</template>
