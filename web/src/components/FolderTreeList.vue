<script setup lang="ts">
import { RouterLink } from 'vue-router'
import type { FolderTreeNode } from '@/api/types'

defineProps<{
  nodes: FolderTreeNode[]
  depth?: number
}>()
</script>

<template>
  <ul class="space-y-0.5 text-sm">
    <li v-for="node in nodes" :key="node.id">
      <RouterLink
        :to="{
          name: 'folder',
          params: {
            id: String(node.id)
          }
        }"
        class="block rounded-md py-1.5 pr-2 hover:bg-moss-soft/60"
        :style="{ paddingLeft: `${(depth ?? 0) * 12 + 8}px` }"
        active-class="bg-moss-soft text-moss-dark"
      >
        {{ node.name }}
      </RouterLink>
      <FolderTreeList
        v-if="node.children?.length"
        :nodes="node.children"
        :depth="(depth ?? 0) + 1"
      />
    </li>
  </ul>
</template>
