<script setup lang="ts">
import { provide } from 'vue'
import { RouterView } from 'vue-router'
import AppFooter from '@/components/AppFooter.vue'
import AppHeader from '@/components/AppHeader.vue'

provide('reloadSidebar', () => {})
</script>

<template>
  <div class="flex min-h-screen flex-col bg-white">
    <AppHeader />
    <main class="px-[0.2em] pt-[0.4em] pb-[2em] min-w-0 flex-1">
      <RouterView />
    </main>
    <AppFooter />
  </div>
</template>
