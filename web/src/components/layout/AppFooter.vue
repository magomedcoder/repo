<script setup lang="ts">
import LocaleSwitch from '@/components/LocaleSwitch.vue'
</script>

<template>
  <footer class="glass mt-auto flex flex-wrap items-center justify-between gap-x-3 gap-y-1.5 px-2.5 py-2 text-[90%] text-ink-muted sm:px-4">
    <p class="m-0 min-w-0">
      {{ $t('footer.poweredBy') }}
      <a
        href="https://magomedcoder.ru/repo"
        class="text-ink-muted no-underline hover:text-ink hover:underline"
        target="_blank"
        rel="noopener noreferrer"
      >Repo</a>
    </p>
    <LocaleSwitch />
  </footer>
</template>
