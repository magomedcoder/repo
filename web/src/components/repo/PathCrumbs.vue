<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import type { RouteLocationRaw } from 'vue-router'
import Icon from '@/components/ui/Icon.vue'
import { repoLocation } from '@/lib/repoNav'

const props = defineProps<{
  repoPath: string
  refName: string
  path?: string
}>()

type Crumb = {
  label: string
  to: RouteLocationRaw | null
  branch?: boolean
}

const crumbs = computed((): Crumb[] => {
  const parts = (props.path || '').split('/').filter(Boolean)
  const items: Crumb[] = []

  items.push({
    label: props.refName,
    branch: true,
    to: parts.length
      ? repoLocation('repo', props.repoPath, { 
          query: { 
            ref: props.refName 
          } 
        })
      : null,
  })

  let acc = ''
  for (let i = 0; i < parts.length; i++) {
    const part = parts[i]!
    acc = acc ? `${acc}/${part}` : part
    const isLast = i === parts.length - 1
    items.push({
      label: part,
      to: isLast
        ? null
        : repoLocation('repo', props.repoPath, {
            query: { 
              ref: props.refName, 
              path: acc 
            },
          }),
    })
  }

  return items
})
</script>

<template>
  <nav
    class="flex min-w-0 flex-wrap items-center gap-x-1 text-[12px] leading-none"
    :aria-label="$t('index.tree')"
  >
    <template v-for="(crumb, i) in crumbs" :key="`${i}-${crumb.label}`">
      <span v-if="i" class="select-none text-ink-muted">/</span>
      <RouterLink
        v-if="crumb.to && crumb.branch"
        :to="crumb.to"
        class="btn !h-8 !min-h-8 !gap-1.5 !px-2.5 !py-0 !text-[10pt] !leading-none font-semibold no-underline hover:no-underline"
      >
        <Icon name="branch" size="sm" class="text-ink-muted" />
        {{ crumb.label }}
      </RouterLink>
      <span
        v-else-if="crumb.branch"
        class="btn !h-8 !min-h-8 !gap-1.5 !px-2.5 !py-0 !text-[10pt] !leading-none font-semibold"
      >
        <Icon name="branch" size="sm" class="text-ink-muted" />
        {{ crumb.label }}
      </span>
      <RouterLink
        v-else-if="crumb.to"
        :to="crumb.to"
        class="font-semibold text-link no-underline hover:underline"
      >
        {{ crumb.label }}
      </RouterLink>
      <span v-else class="font-semibold text-ink">{{ crumb.label }}</span>
    </template>
  </nav>
</template>
