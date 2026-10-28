<script setup lang="ts">
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
  <article class="overflow-hidden border border-line bg-paper">
    <header class="flex items-center gap-2 border-b border-line bg-path px-3 py-1.5 text-[10pt] text-ink">{{ readme.path }}</header>
    <div :class="['[&_h1]:mb-[0.4em] [&_h1]:mt-[1em] [&_h1]:text-[140%] [&_h1]:font-medium [&_h1]:text-ink [&_h2]:mb-[0.35em] [&_h2]:mt-[0.9em] [&_h2]:text-[120%] [&_h2]:font-medium [&_h2]:text-ink [&_h3]:mb-[0.3em] [&_h3]:mt-[0.8em] [&_h3]:font-medium [&_h3]:text-ink [&_p]:my-[0.5em] [&_ul]:my-[0.5em] [&_ul]:list-disc [&_ul]:pl-[1.4em] [&_ol]:my-[0.5em] [&_ol]:list-decimal [&_ol]:pl-[1.4em] [&_pre]:my-[0.6em] [&_pre]:overflow-x-auto [&_pre]:border [&_pre]:border-line [&_pre]:bg-paper-2 [&_pre]:p-2 [&_pre]:font-mono [&_pre]:text-[90%] [&_code]:font-mono [&_code]:text-[90%] [&_a]:text-link [&_blockquote]:my-[0.5em] [&_blockquote]:border-l [&_blockquote]:border-line [&_blockquote]:pl-3 [&_blockquote]:text-ink-muted', 'px-4 py-3 text-sm leading-relaxed sm:px-6']" v-html="html" />
  </article>
</template>
