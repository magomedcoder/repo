<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import type { RouteLocationRaw } from 'vue-router'
import type { TreeEntry } from '@/api/types'
import PathCrumbs from '@/components/repo/PathCrumbs.vue'
import Icon from '@/components/ui/Icon.vue'
import { formatBytes } from '@/lib/bytes'
import { idleAge } from '@/lib/idle'
import { formatDate } from '@/i18n'
import { entryRoute } from '@/lib/repoNav'

const props = defineProps<{
  entries: TreeEntry[]
  repoPath: string
  refName: string
  path?: string
  parentTo?: RouteLocationRaw | null
}>()

const sorted = computed(() =>
  [...props.entries].sort((a, b) => {
    if (a.type !== b.type) {
      return a.type === 'tree' ? -1 : 1
    }

    return a.name.localeCompare(b.name)
  }),
)

function modifiedLabel(entry: TreeEntry) {
  return idleAge(entry.modified_at) || '-'
}
</script>

<template>
  <div class="surface overflow-hidden">
    <div class="surface-header flex-wrap items-center gap-x-3 gap-y-1.5 py-2">
      <PathCrumbs
        class="min-w-0 flex-1"
        :repo-path="repoPath"
        :ref-name="refName"
        :path="path"
      />
      <div v-if="$slots.actions" class="flex h-8 shrink-0 items-center gap-1.5">
        <slot name="actions" />
      </div>
    </div>

    <slot name="banner" />

    <div class="overflow-x-auto">
      <table class="w-full min-w-[20rem] table-fixed border-collapse text-[12px] leading-snug">
        <colgroup>
          <col>
          <col class="w-[7rem]">
        </colgroup>
        <tbody>
          <tr v-if="parentTo" class="border-b border-line/80 hover:bg-row-hover">
            <td class="px-3 py-1.5" colspan="2">
              <RouterLink :to="parentTo" class="text-ink-muted no-underline hover:text-link hover:underline">
                ..
              </RouterLink>
            </td>
          </tr>
          <tr
            v-for="entry in sorted"
            :key="entry.path"
            class="border-b border-line/80 last:border-b-0 hover:bg-row-hover"
          >
            <td class="min-w-0 truncate px-3 py-1.5">
              <RouterLink
                :to="entryRoute(repoPath, refName, entry)"
                class="inline-flex max-w-full items-center gap-2 text-ink no-underline hover:text-link hover:underline"
              >
                <Icon
                  :name="entry.type === 'tree' ? 'folder' : 'file'"
                  size="sm"
                  class="text-ink-muted"
                />
                <span
                  class="min-w-0 truncate"
                  :class="entry.type === 'tree' ? 'font-semibold' : 'font-normal'"
                >
                  {{ entry.type === 'tree' ? `${entry.name}/` : entry.name }}
                </span>
                <span
                  v-if="entry.type !== 'tree' && entry.size"
                  class="shrink-0 text-[11px] tabular-nums text-ink-muted no-underline"
                >
                  {{ formatBytes(entry.size) }}
                </span>
              </RouterLink>
            </td>
            <td
              class="whitespace-nowrap px-3 py-1.5 text-right tabular-nums text-ink-muted"
              :title="entry.modified_at ? formatDate(entry.modified_at) : undefined"
            >
              {{ modifiedLabel(entry) }}
            </td>
          </tr>
          <tr v-if="!sorted.length">
            <td colspan="2" class="px-3 py-3 text-ink-muted">
              {{ $t('repo.empty') }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
