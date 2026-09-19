<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { setLocale, type AppLocale } from '@/i18n'

defineProps<{
  inverted?: boolean
}>()

const { locale, t } = useI18n()

function pick(next: AppLocale) {
  setLocale(next)
}
</script>

<template>
  <div
    class="inline-flex overflow-hidden rounded-md border border-line text-xs font-medium"
    :class="inverted ? 'border-white/30' : ''"
    role="group"
    :aria-label="t('nav.language')"
  >
    <button
      type="button"
      class="px-2 py-1"
      :class="locale === 'en'
        ? (inverted ? 'bg-white text-header' : 'bg-paper-2 text-ink')
        : (inverted ? 'text-white/80 hover:bg-white/10' : 'bg-white text-ink-muted hover:bg-paper')"
      @click="pick('en')"
    >
      EN
    </button>
    <button
      type="button"
      class="px-2 py-1"
      :class="locale === 'ru'
        ? (inverted ? 'bg-white text-header' : 'bg-paper-2 text-ink')
        : (inverted ? 'text-white/80 hover:bg-white/10' : 'bg-white text-ink-muted hover:bg-paper')"
      @click="pick('ru')"
    >
      RU
    </button>
  </div>
</template>
