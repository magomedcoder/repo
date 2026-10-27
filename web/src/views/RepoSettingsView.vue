<script setup lang="ts">
import RepoShell from '@/components/repo/RepoShell.vue'
import RepoSettingsBody from '@/views/repo/RepoSettingsBody.vue'

defineProps<{
  repoPath: string
}>()
</script>

<template>
  <RepoShell :repo-path="repoPath" tab="settings">
    <RepoSettingsBody :repo-path="repoPath" />
  </RepoShell>
</template>
