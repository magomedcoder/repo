<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import Icon from '@/components/ui/Icon.vue'
import type { IconName } from '@/components/ui/Icon.vue'

const items: { name: string; labelKey: string; icon: IconName }[] = [
  { name: 'cp', labelKey: 'cp.nav.overview', icon: 'home' },
  { name: 'users', labelKey: 'cp.nav.users', icon: 'users' },
  { name: 'cp-repos', labelKey: 'cp.nav.repos', icon: 'repo' },
  { name: 'access-logs', labelKey: 'cp.nav.accessLogs', icon: 'list' },
  { name: 'admin-mirrors', labelKey: 'cp.nav.mirrors', icon: 'mirror' },
]

const route = useRoute()
const active = computed(() => String(route.name || ''))

function navClass(name: string) {
  const isActive = active.value === name
  return [
    'inline-flex shrink-0 items-center gap-1.5 rounded-md px-2.5 py-1.5 text-[12px] no-underline hover:no-underline md:w-full',
    isActive
      ? 'bg-white/65 font-semibold text-ink shadow-sm'
      : 'font-medium text-ink-muted hover:bg-white/40 hover:text-ink',
  ]
}
</script>

<template>
  <nav
    class="glass flex shrink-0 gap-1 overflow-x-auto p-1.5 md:sticky md:top-3 md:w-48 md:flex-col md:overflow-visible"
    :aria-label="$t('nav.controlPanel')"
  >
    <RouterLink
      v-for="item in items"
      :key="item.name"
      :to="{ name: item.name }"
      :class="navClass(item.name)"
    >
      <Icon :name="item.icon" size="sm" class="opacity-80" />
      {{ $t(item.labelKey) }}
    </RouterLink>
  </nav>
</template>
