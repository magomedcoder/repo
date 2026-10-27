<script setup lang="ts">
import RepoShell from '@/components/repo/RepoShell.vue'
import RepoAboutBody from '@/views/repo/RepoAboutBody.vue'

defineProps<{
  repoPath: string
}>()
</script>

<template>
  <RepoShell :repo-path="repoPath" tab="about">
    <RepoAboutBody :repo-path="repoPath" />
  </RepoShell>
</template>
