<script setup lang="ts">
import { provide, ref } from 'vue'
import { RouterView } from 'vue-router'
import AppHeader from '@/components/AppHeader.vue'
import FolderSidebar from '@/components/FolderSidebar.vue'

const sidebarRef = ref<{ reload: () => void } | null>(null)
provide('reloadSidebar', () => sidebarRef.value?.reload())
</script>

<template>
  <div class="min-h-screen">
    <AppHeader />
    <div class="mx-auto flex max-w-6xl gap-6 px-4 py-6">
      <FolderSidebar ref="sidebarRef" />
      <main class="min-w-0 flex-1">
        <RouterView />
      </main>
    </div>
  </div>
</template>
