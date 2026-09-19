<script setup lang="ts">
import { RouterLink, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { useBreadcrumbs } from '@/composables/useBreadcrumbs'
import Icon from '@/components/ui/Icon.vue'
import LocaleSwitch from '@/components/LocaleSwitch.vue'

const auth = useAuth()
const { crumbs } = useBreadcrumbs()
const router = useRouter()

async function onLogout() {
  await auth.logout()
  await router.push({
    name: 'login'
  })
}
</script>

<template>
  <header class="bg-header text-white">
    <div class="mx-auto flex h-14 max-w-7xl items-center gap-4 px-4">
      <RouterLink to="/" class="flex items-center gap-2 font-semibold text-white">
        <Icon name="mark" />
        Repo
      </RouterLink>

      <nav
        v-if="crumbs.length"
        class="hidden min-w-0 flex-1 items-center gap-1.5 overflow-x-auto text-sm text-white/70 sm:flex"
        :aria-label="$t('nav.breadcrumb')"
      >
        <template
          v-for="(crumb, i) in crumbs"
          :key="`${crumb.labelKey || crumb.label}-${i}`"
        >
          <span v-if="i > 0">/</span>
          <RouterLink
            v-if="crumb.to"
            :to="crumb.to"
            class="truncate text-white hover:underline"
          >
            {{ crumb.labelKey ? $t(crumb.labelKey) : crumb.label }}
          </RouterLink>
          <span v-else class="truncate text-white">{{ crumb.labelKey ? $t(crumb.labelKey) : crumb.label }}</span>
        </template>
      </nav>
      <div v-else class="flex-1" />

      <LocaleSwitch inverted />
      <span class="hidden text-sm text-white/80 sm:inline">{{ auth.user.value?.username }}</span>
      <button
        type="button" class="text-sm text-white/80 hover:text-white"
        @click="onLogout"
      >
        {{ $t('nav.logout') }}
      </button>
    </div>
  </header>
</template>
