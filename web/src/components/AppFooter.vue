<script setup lang="ts">
import LocaleSwitch from '@/components/LocaleSwitch.vue'
</script>

<template>
  <footer class="mt-auto flex items-center justify-between gap-3 border-t border-line py-2 text-[90%] text-ink-muted">
    <p class="m-0">
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
