<script setup lang="ts">
import { computed } from 'vue'
import close from '@/assets/icons/close.svg?raw'
import issue from '@/assets/icons/issue.svg?raw'
import issueClosed from '@/assets/icons/issue-closed.svg?raw'
import lock from '@/assets/icons/lock.svg?raw'
import pull from '@/assets/icons/pull.svg?raw'
import pullMerged from '@/assets/icons/pull-merged.svg?raw'

type IconName = 'close' | 'issue' | 'issue-closed' | 'lock' | 'pull' | 'pull-merged'
type IconSize = 'xs' | 'sm' | 'md'

const icons: Record<IconName, string> = {
  close,
  issue,
  'issue-closed': issueClosed,
  lock,
  pull,
  'pull-merged': pullMerged,
}

const sizeClass: Record<IconSize, string> = {
  xs: 'h-2.5 w-2.5',
  sm: 'h-3.5 w-3.5',
  md: 'h-4 w-4',
}

const props = withDefaults(
  defineProps<{
    name: IconName
    size?: IconSize
  }>(),
  {
    size: 'md',
  },
)

const svg = computed(() => icons[props.name])
</script>

<template>
  <span
    class="inline-block shrink-0 [&>svg]:block [&>svg]:h-full [&>svg]:w-full"
    :class="sizeClass[size]"
    v-html="svg"
  />
</template>
