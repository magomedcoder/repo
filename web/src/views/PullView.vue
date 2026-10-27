<script setup lang="ts">
import RepoShell from '@/components/repo/RepoShell.vue'
import PullBody from '@/views/repo/PullBody.vue'

defineProps<{
  repoPath: string
  number: string
}>()
</script>

<template>
  <RepoShell :repo-path="repoPath" tab="pulls">
    <PullBody :repo-path="repoPath" :number="number" />
  </RepoShell>
</template>
