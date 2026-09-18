<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { setLocale, type AppLocale } from '@/i18n'

const { locale, t } = useI18n()

function pick(next: AppLocale) {
  setLocale(next)
}
</script>

<template>
  <div class="inline-flex overflow-hidden rounded-md border border-line text-xs font-semibold" role="group" :aria-label="t('nav.language')">
    <button
      type="button"
      class="px-2 py-1 transition"
      :class="locale === 'en' ? 'bg-moss text-white' : 'bg-white/70 text-ink hover:bg-paper-2'"
      @click="pick('en')"
    >
      EN
    </button>
    <button
      type="button"
      class="px-2 py-1 transition"
      :class="locale === 'ru' ? 'bg-moss text-white' : 'bg-white/70 text-ink hover:bg-paper-2'"
      @click="pick('ru')"
    >
      RU
    </button>
  </div>
</template>
