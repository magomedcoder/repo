<script setup lang="ts">
import { RouterLink } from 'vue-router'
import LocaleSwitch from '@/components/LocaleSwitch.vue'
</script>

<template>
  <footer class="mt-auto flex items-center justify-between gap-3 border-t border-line py-2 text-[90%] text-ink-muted">
    <RouterLink
      to="/"
      class="text-ink-muted no-underline hover:text-ink hover:no-underline"
    >
      {{ $t('brand.name') }}
    </RouterLink>
    <LocaleSwitch />
  </footer>
</template>
