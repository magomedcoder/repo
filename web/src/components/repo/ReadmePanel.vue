<script setup lang="ts">
import { computed } from 'vue'
import { marked } from 'marked'
import type { BlobContent } from '@/api/types'

const props = defineProps<{
  readme: BlobContent
}>()

const html = computed(() => {
  if (!props.readme.content || props.readme.is_binary) {
    return ''
  }

  return marked.parse(props.readme.content, { async: false }) as string
})
</script>

<template>
  <article class="overflow-hidden rounded-md border border-line bg-white">
    <header class="border-b border-line bg-paper px-4 py-2 font-mono text-sm font-semibold">{{ readme.path }}</header>
    <div class="markdown-body px-6 py-4 text-sm leading-relaxed" v-html="html" />
  </article>
</template>
