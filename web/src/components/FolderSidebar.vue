<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { foldersApi } from '@/api'
import FolderTreeList from '@/components/FolderTreeList.vue'
import type { FolderTreeNode } from '@/api/types'

const tree = ref<FolderTreeNode[]>([])

async function load() {
  try {
    const res = await foldersApi.tree()
    tree.value = res.tree ?? []
  } catch {
    tree.value = []
  }
}

onMounted(load)
defineExpose({ reload: load })
</script>

<template>
  <aside class="panel sticky top-4 hidden h-fit w-56 shrink-0 p-3 lg:block">
    <p class="mb-2 px-2 text-xs font-semibold uppercase tracking-wide text-ink-muted">{{ $t('nav.folders') }}</p>
    <ul class="space-y-0.5 text-sm">
      <li>
        <RouterLink
          to="/"
          class="block rounded-md px-2 py-1.5 hover:bg-moss-soft/60"
          active-class="bg-moss-soft text-moss-dark"
        >
          {{ $t('nav.root') }}
        </RouterLink>
      </li>
    </ul>
    <FolderTreeList :nodes="tree" />
  </aside>
</template>
