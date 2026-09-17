<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { useBreadcrumbs } from '@/composables/useBreadcrumbs'
import { useRouter } from 'vue-router'

const auth = useAuth()
const { crumbs } = useBreadcrumbs()
const router = useRouter()

async function onLogout() {
  await auth.logout()
  router.push({ name: 'login' })
}
</script>

<template>
  <header class="border-b border-line/80 bg-white/70 backdrop-blur-md">
    <div class="mx-auto flex max-w-6xl items-center gap-4 px-4 py-3">
      <RouterLink to="/" class="font-display text-xl font-extrabold tracking-tight text-ink">
        Repo
      </RouterLink>

      <nav
        v-if="crumbs.length"
        class="hidden min-w-0 flex-1 items-center gap-1.5 overflow-x-auto font-mono text-sm text-ink-muted sm:flex"
        aria-label="Breadcrumb"
      >
        <template v-for="(crumb, i) in crumbs" :key="`${crumb.label}-${i}`">
          <span v-if="i > 0" class="text-line">/</span>
          <RouterLink v-if="crumb.to" :to="crumb.to" class="truncate text-ink hover:text-moss">
            {{ crumb.label }}
          </RouterLink>
          <span v-else class="truncate text-ink">{{ crumb.label }}</span>
        </template>
      </nav>
      <div v-else class="flex-1" />

      <div class="flex items-center gap-3 text-sm">
        <span class="hidden font-mono text-ink-muted sm:inline">{{ auth.user.value?.username }}</span>
        <button
          type="button"
          class="btn-ghost"
          @click="onLogout"
        >Log out</button>
      </div>
    </div>
  </header>
</template>
