<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import type { RouteLocationRaw } from 'vue-router'
import { repoLocation } from '@/lib/repoNav'

const props = defineProps<{
  repoPath: string
  refName: string
  path?: string
}>()

type Crumb = {
  label: string
  to: RouteLocationRaw | null
}

const crumbs = computed((): Crumb[] => {
  const parts = (props.path || '').split('/').filter(Boolean)
  const items: Crumb[] = []

  if (!parts.length) {
    items.push({ 
      label: props.refName, 
      to: null
    })
    return items
  }

  items.push({
    label: props.refName,
    to: repoLocation('repo-tree', props.repoPath, {
      query: { ref: props.refName },
    }),
  })

  let acc = ''
  for (let i = 0; i < parts.length; i++) {
    const part = parts[i]!
    acc = acc ? `${acc}/${part}` : part
    const isLast = i === parts.length - 1
    if (isLast) {
      items.push({ label: part, to: null })
      continue
    }

    items.push({
      label: part,
      to: repoLocation('repo-tree', props.repoPath, {
        query: {
          ref: props.refName,
          path: acc,
        },
      }),
    })
  }

  return items
})
</script>

<template>
  <nav
    class="flex min-w-0 flex-wrap items-baseline gap-x-1 font-mono text-[90%]"
    :aria-label="$t('index.tree')"
  >
    <template v-for="(crumb, i) in crumbs" :key="`${i}-${crumb.label}`">
      <span v-if="i" class="text-ink-muted">/</span>
      <RouterLink
        v-if="crumb.to"
        :to="crumb.to"
        class="text-ink-muted no-underline hover:text-ink hover:underline"
      >
        {{ crumb.label }}
      </RouterLink>
      <span v-else class="font-bold text-ink">{{ crumb.label }}</span>
    </template>
  </nav>
</template>
