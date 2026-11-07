<script setup lang="ts">
import { computed } from 'vue'
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import type { BlobContent } from '@/api/types'
import Icon from '@/components/ui/Icon.vue'

const props = defineProps<{
  readme: BlobContent
}>()

const ALLOWED_URI_REGEXP = /^(?:\/(?!\/)|https?:\/\/|#|mailto:)/i

function isExternalUrl(url: string): boolean {
  return /^(?:https?:)?\/\//i.test(url)
}

function isBlockedImageUrl(url: string): boolean {
  const trimmed = url.trim()
  if (!trimmed || trimmed.startsWith('#') || trimmed.startsWith('mailto:')) {
    return true
  }

  if (isExternalUrl(trimmed)) {
    return true
  }

  return trimmed.includes('://')
}

function sanitizeReadmeHtml(raw: string): string {
  DOMPurify.addHook('afterSanitizeAttributes', (node) => {
    if (node.tagName === 'A') {
      const href = node.getAttribute('href')
      if (href && isExternalUrl(href)) {
        node.setAttribute('rel', 'noopener noreferrer')
        if (!node.getAttribute('target')) {
          node.setAttribute('target', '_blank')
        }
      }
      if (node.getAttribute('target') === '_blank') {
        node.setAttribute('rel', 'noopener noreferrer')
      }
    }

    if (node.tagName === 'IMG') {
      const src = node.getAttribute('src')
      if (src && isBlockedImageUrl(src)) {
        node.removeAttribute('src')
        node.setAttribute('alt', node.getAttribute('alt') || '')
      }
    }
  })

  try {
    return DOMPurify.sanitize(raw, {
      FORBID_ATTR: ['style'],
      ALLOWED_URI_REGEXP,
    })
  } finally {
    DOMPurify.removeHook('afterSanitizeAttributes')
  }
}

const html = computed(() => {
  if (!props.readme.content || props.readme.is_binary) {
    return ''
  }

  const raw = marked.parse(props.readme.content, { async: false }) as string
  return sanitizeReadmeHtml(raw)
})
</script>

<template>
  <article class="surface">
    <header class="surface-header py-2 text-[12px] font-semibold">
      <Icon name="book" size="sm" class="text-ink-muted" />
      {{ readme.path }}
    </header>
    <div :class="['[&_h1]:mb-[0.4em] [&_h1]:mt-[1em] [&_h1]:text-[140%] [&_h1]:font-medium [&_h1]:text-ink [&_h2]:mb-[0.35em] [&_h2]:mt-[0.9em] [&_h2]:text-[120%] [&_h2]:font-medium [&_h2]:text-ink [&_h3]:mb-[0.3em] [&_h3]:mt-[0.8em] [&_h3]:font-medium [&_h3]:text-ink [&_p]:my-[0.5em] [&_ul]:my-[0.5em] [&_ul]:list-disc [&_ul]:pl-[1.4em] [&_ol]:my-[0.5em] [&_ol]:list-decimal [&_ol]:pl-[1.4em] [&_pre]:my-[0.6em] [&_pre]:overflow-x-auto [&_pre]:border [&_pre]:border-line [&_pre]:bg-paper-2 [&_pre]:p-2 [&_pre]:font-mono [&_pre]:text-[90%] [&_code]:font-mono [&_code]:text-[90%] [&_a]:text-link [&_blockquote]:my-[0.5em] [&_blockquote]:border-l [&_blockquote]:border-line [&_blockquote]:pl-3 [&_blockquote]:text-ink-muted [&_img]:inline [&_img]:max-w-full [&_img]:align-text-bottom', 'px-4 py-4 text-sm leading-relaxed sm:px-6']" v-html="html" />
  </article>
</template>
