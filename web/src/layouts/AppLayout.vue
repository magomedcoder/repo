<script setup lang="ts">
import { computed, provide, ref } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import AppHeader from '@/components/AppHeader.vue'
import FolderSidebar from '@/components/FolderSidebar.vue'

const route = useRoute()
const sidebarRef = ref<{ reload: () => void } | null>(null)
provide('reloadSidebar', () => sidebarRef.value?.reload())

const showSidebar = computed(() => route.name === 'home' || route.name === 'folder')
</script>

<template>
  <div class="min-h-screen bg-paper">
    <AppHeader />
    <div v-if="showSidebar" class="mx-auto flex max-w-7xl gap-6 px-4 py-6">
      <FolderSidebar ref="sidebarRef" />
      <main class="min-w-0 flex-1">
        <RouterView />
      </main>
    </div>
    <RouterView v-else />
  </div>
</template>
