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
  <aside class="sticky top-4 hidden h-fit w-56 shrink-0 lg:block">
    <p class="mb-2 px-2 text-xs font-semibold text-ink">{{ $t('nav.folders') }}</p>
    <ul class="space-y-0.5 text-sm">
      <li>
        <RouterLink
          to="/"
          class="block rounded-md px-2 py-1.5 text-ink hover:bg-paper-2"
          active-class="bg-paper-2 font-semibold"
        >
          {{ $t('nav.root') }}
        </RouterLink>
      </li>
    </ul>
    <FolderTreeList :nodes="tree" />
  </aside>
</template>
