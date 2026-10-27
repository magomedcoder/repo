<script setup lang="ts">
import { computed } from 'vue'
import close from '@/assets/icons/close.svg?raw'
import issue from '@/assets/icons/issue.svg?raw'
import issueClosed from '@/assets/icons/issue-closed.svg?raw'
import pull from '@/assets/icons/pull.svg?raw'
import pullMerged from '@/assets/icons/pull-merged.svg?raw'

type IconName = 'close' | 'issue' | 'issue-closed' | 'pull' | 'pull-merged'

const icons: Record<IconName, string> = {
  close,
  issue,
  'issue-closed': issueClosed,
  pull,
  'pull-merged': pullMerged,
}

const props = defineProps<{
  name: IconName
}>()

const svg = computed(() => icons[props.name])
</script>

<template>
  <span
    class="inline-block h-4 w-4 shrink-0 [&>svg]:block [&>svg]:h-full [&>svg]:w-full"
    v-html="svg"
  />
</template>
