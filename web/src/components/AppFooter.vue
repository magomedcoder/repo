<script setup lang="ts">
import LocaleSwitch from '@/components/LocaleSwitch.vue'
</script>

<template>
  <footer class="border-t border-line bg-white px-3 py-2 text-[13px] text-ink-muted">
    <div class="flex items-center justify-between gap-3">
      <a href="https://git.magomedcoder.ru/repo">Repo</a>
      <LocaleSwitch />
    </div>
  </footer>
</template>
