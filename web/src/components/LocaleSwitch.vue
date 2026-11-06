<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { setLocale, type AppLocale } from '@/i18n'

const { locale, t } = useI18n()

function pick(next: AppLocale) {
  setLocale(next)
}
</script>

<template>
  <span role="group" class="inline-flex items-center gap-1" :aria-label="t('nav.language')">
    <button
      type="button"
      class="cursor-pointer border-0 bg-transparent p-0 text-[90%] uppercase"
      :class="locale === 'ru' ? 'font-medium text-ink' : 'text-ink-muted hover:text-ink'"
      :aria-pressed="locale === 'ru'"
      @click="pick('ru')"
    >
      {{ t('nav.langRu') }}
    </button>

    <button
      type="button"
      class="cursor-pointer border-0 bg-transparent p-0 text-[90%] uppercase"
      :class="locale === 'en' ? 'font-medium text-ink' : 'text-ink-muted hover:text-ink'"
      :aria-pressed="locale === 'en'"
      @click="pick('en')"
    >
      {{ t('nav.langEn') }}
    </button>
  </span>
</template>
