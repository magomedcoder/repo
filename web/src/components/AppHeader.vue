<script setup lang="ts">
import { computed, inject, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { useSite } from '@/composables/useSite'

const auth = useAuth()
const { name: siteName } = useSite()
const route = useRoute()
const router = useRouter()
const searchQ = ref(typeof route.query.q === 'string' ? route.query.q : '')

const openNewFolder = inject<() => void>('openNewFolder', () => {})
const openNewRepo = inject<() => void>('openNewRepo', () => {})

const username = computed(() => auth.user.value?.username || '')
const isAdmin = computed(() => auth.isAdmin.value)

watch(
  () => route.query.q,
  (v) => {
    searchQ.value = typeof v === 'string' ? v : ''
  },
)

async function onSearch() {
  const q = searchQ.value.trim()
  await router.push({
    name: 'search',
    query: q ? { q } : {},
  })
}

async function onLogout() {
  await auth.logout()
  await router.push({ name: 'login' })
}
</script>

<template>
  <header class="mb-4 flex flex-wrap items-center justify-between gap-x-3 gap-y-1.5 border-b border-line py-2">
    <div class="flex min-w-0 items-baseline gap-2.5">
      <RouterLink
        to="/"
        class="text-[1.35rem] font-medium leading-none tracking-tight text-ink no-underline hover:no-underline"
      >
        {{ siteName }}
      </RouterLink>
      <span class="hidden truncate text-[85%] text-ink-muted md:inline">{{ $t('nav.tagline') }}</span>
    </div>
    <nav class="flex max-w-full flex-wrap items-center justify-end gap-x-2.5 gap-y-1 text-[90%]">
      <template v-if="username">
        <RouterLink :to="{ name: 'folders' }">{{ $t('nav.folders') }}</RouterLink>
        <button
          type="button"
          class="cursor-pointer border-0 bg-transparent p-0 text-link hover:underline"
          @click="openNewFolder"
        >
          {{ $t('home.newFolder') }}
        </button>
        <button
          type="button"
          class="cursor-pointer border-0 bg-transparent p-0 text-link hover:underline"
          @click="openNewRepo"
        >
          {{ $t('home.newRepository') }}
        </button>
      </template>
      <form class="contents" @submit.prevent="onSearch">
        <label class="sr-only" for="global-search">{{ $t('search.title') }}</label>
        <input
          id="global-search"
          v-model="searchQ"
          type="search"
          class="w-28 border border-line bg-paper px-1.5 py-0.5 text-[90%] text-ink outline-none sm:w-40"
          :placeholder="$t('search.placeholder')"
          :aria-label="$t('search.title')"
        >
      </form>
      <template v-if="username">
        <RouterLink v-if="isAdmin" :to="{ name: 'users' }">{{ $t('nav.users') }}</RouterLink>
        <RouterLink :to="{ name: 'profile' }">{{ $t('nav.profile') }}</RouterLink>
        <button
          type="button"
          class="cursor-pointer border-0 bg-transparent p-0 text-link hover:underline"
          @click="onLogout"
        >
          {{ $t('nav.logout') }}
        </button>
      </template>
      <template v-else>
        <RouterLink :to="{ name: 'login' }">{{ $t('auth.signIn') }}</RouterLink>
      </template>
    </nav>
  </header>
</template>
