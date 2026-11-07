<script setup lang="ts">
import LocaleSwitch from '@/components/LocaleSwitch.vue'
import { useSite } from '@/composables/useSite'

const { name: siteName } = useSite()
</script>

<template>
  <footer class="glass mt-auto grid grid-cols-[1fr_auto_1fr] items-center gap-x-3 gap-y-1.5 px-2.5 py-2 text-[90%] text-ink-muted sm:px-4">
    <p class="m-0 min-w-0 justify-self-start">
      {{ $t('footer.copyright', { name: siteName }) }}
    </p>
    <p class="m-0 justify-self-center whitespace-nowrap text-center">
      {{ $t('footer.poweredBy') }}
      <a
        href="https://magomedcoder.ru/repo"
        class="text-ink-muted no-underline hover:text-ink hover:underline"
        target="_blank"
        rel="noopener noreferrer"
      >Repo</a>
    </p>
    <div class="justify-self-end">
      <LocaleSwitch />
    </div>
  </footer>
</template>
