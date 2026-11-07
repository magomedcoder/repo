<script setup lang="ts">
import { RouterLink } from 'vue-router'
import Icon from '@/components/ui/Icon.vue'
import { repoLocation } from '@/lib/repoNav'
import { SETTINGS_SECTIONS, type SettingsSection } from './types'

const props = defineProps<{
  repoPath: string
  section: SettingsSection
}>()

function settingsLocation(path: string, sectionId: SettingsSection = props.section) {
  const loc = repoLocation('repo-settings', path)
  const base = typeof loc === 'string' ? { path: loc } : { ...loc }
  return { 
    ...base,
    hash: `#${sectionId}`
  }
}

function navClass(id: SettingsSection) {
  const active = props.section === id
  return [
    'shrink-0 border-b-2 px-3 py-1.5 text-left text-[10pt] no-underline hover:no-underline md:block md:w-full md:border-b-0 md:border-l-2',
    active
      ? 'border-ink bg-white/55 font-semibold text-ink'
      : 'border-transparent text-ink-muted hover:bg-row-hover hover:text-ink',
  ]
}
</script>

<template>
  <nav
    class="glass flex shrink-0 gap-1 overflow-x-auto px-1 py-1 md:w-44 md:flex-col md:overflow-visible md:px-2 md:py-2"
    :aria-label="$t('settings.title')"
  >
    <RouterLink
      v-for="item in SETTINGS_SECTIONS"
      :key="item.id"
      :to="settingsLocation(repoPath, item.id)"
      :class="navClass(item.id)"
    >
      <span class="inline-flex items-center gap-1.5">
        <Icon :name="item.icon" size="sm" />
        {{ $t(item.labelKey) }}
      </span>
    </RouterLink>
  </nav>
</template>
