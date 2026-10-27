<script setup lang="ts">
import { markdown } from '@/lib/classes'
import { computed } from 'vue'
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import type { BlobContent } from '@/api/types'

const props = defineProps<{
  readme: BlobContent
}>()

const html = computed(() => {
  if (!props.readme.content || props.readme.is_binary) {
    return ''
  }

  const raw = marked.parse(props.readme.content, { async: false }) as string
  return DOMPurify.sanitize(raw)
})
</script>

<template>
  <article class="term-block">
    <header class="term-block-hd">{{ readme.path }}</header>
    <div :class="[markdown, 'px-4 py-3 text-sm leading-relaxed sm:px-6']" v-html="html" />
  </article>
</template>
