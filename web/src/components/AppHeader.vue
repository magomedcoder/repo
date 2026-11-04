<script setup lang="ts">
import { computed, inject, onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { usersApi } from '@/api'
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
const avatarSrc = computed(() => {
  const u = auth.user.value
  if (!u?.has_avatar || !u.username) {
    return ''
  }
  return usersApi.avatarURL(u.username)
})
const avatarInitial = computed(() => (username.value.slice(0, 1) || '?').toUpperCase())

const createOpen = ref(false)
const createRoot = ref<HTMLElement | null>(null)
const menuOpen = ref(false)
const menuRoot = ref<HTMLElement | null>(null)

watch(
  () => route.query.q,
  (v) => {
    searchQ.value = typeof v === 'string' ? v : ''
  },
)

watch(
  () => route.fullPath,
  () => {
    createOpen.value = false
    menuOpen.value = false
  },
)

function closeMenus() {
  createOpen.value = false
  menuOpen.value = false
}

function onDocPointerDown(ev: PointerEvent) {
  if (!(ev.target instanceof Node)) {
    return
  }
  if (createOpen.value && createRoot.value && !createRoot.value.contains(ev.target)) {
    createOpen.value = false
  }
  if (menuOpen.value && menuRoot.value && !menuRoot.value.contains(ev.target)) {
    menuOpen.value = false
  }
}

function onDocKeydown(ev: KeyboardEvent) {
  if (ev.key === 'Escape') {
    closeMenus()
  }
}

onMounted(() => {
  document.addEventListener('pointerdown', onDocPointerDown)
  document.addEventListener('keydown', onDocKeydown)
})

onUnmounted(() => {
  document.removeEventListener('pointerdown', onDocPointerDown)
  document.removeEventListener('keydown', onDocKeydown)
})

function toggleCreate() {
  menuOpen.value = false
  createOpen.value = !createOpen.value
}

function toggleMenu() {
  createOpen.value = false
  menuOpen.value = !menuOpen.value
}

function onNewFolder() {
  closeMenus()
  openNewFolder()
}

function onNewRepo() {
  closeMenus()
  openNewRepo()
}

async function onSearch() {
  const q = searchQ.value.trim()
  await router.push({
    name: 'search',
    query: q ? { q } : {},
  })
}

async function onLogout() {
  closeMenus()
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
      <div v-if="username" ref="createRoot" class="relative">
        <button
          type="button"
          class="flex h-7 w-7 cursor-pointer items-center justify-center border border-line bg-paper p-0 text-[18px] leading-none text-ink hover:bg-row-hover"
          :aria-expanded="createOpen"
          aria-haspopup="menu"
          :aria-label="$t('nav.create')"
          @click="toggleCreate"
        >
          <span aria-hidden="true">+</span>
        </button>
        <div
          v-if="createOpen"
          role="menu"
          class="absolute right-0 z-20 mt-1 min-w-44 border border-line bg-paper py-1 shadow-sm"
        >
          <button
            type="button"
            role="menuitem"
            class="block w-full cursor-pointer border-0 bg-transparent px-3 py-1.5 text-left text-ink hover:bg-path"
            @click="onNewFolder"
          >
            {{ $t('home.newFolder') }}
          </button>
          <button
            type="button"
            role="menuitem"
            class="block w-full cursor-pointer border-0 bg-transparent px-3 py-1.5 text-left text-ink hover:bg-path"
            @click="onNewRepo"
          >
            {{ $t('home.newRepository') }}
          </button>
        </div>
      </div>
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
      <div v-if="username" ref="menuRoot" class="relative">
        <button
          type="button"
          class="flex h-7 w-7 cursor-pointer items-center justify-center overflow-hidden rounded-full border border-line bg-paper-2 p-0 text-[11px] font-semibold text-ink hover:bg-row-hover"
          :aria-expanded="menuOpen"
          aria-haspopup="menu"
          :aria-label="username"
          @click="toggleMenu"
        >
          <img
            v-if="avatarSrc"
            :src="avatarSrc"
            alt=""
            class="h-full w-full object-cover"
          >
          <span v-else>{{ avatarInitial }}</span>
        </button>
        <div
          v-if="menuOpen"
          role="menu"
          class="absolute right-0 z-20 mt-1 min-w-44 border border-line bg-paper py-1 shadow-sm"
        >
          <RouterLink
            role="menuitem"
            class="block px-3 py-1.5 text-ink no-underline hover:bg-path hover:no-underline"
            :to="{ name: 'profile' }"
            @click="closeMenus"
          >
            {{ $t('nav.profile') }}
          </RouterLink>
          <RouterLink
            v-if="isAdmin"
            role="menuitem"
            class="block px-3 py-1.5 text-ink no-underline hover:bg-path hover:no-underline"
            :to="{ name: 'access-logs' }"
            @click="closeMenus"
          >
            {{ $t('nav.accessLogs') }}
          </RouterLink>
          <RouterLink
            v-if="isAdmin"
            role="menuitem"
            class="block px-3 py-1.5 text-ink no-underline hover:bg-path hover:no-underline"
            :to="{ name: 'users' }"
            @click="closeMenus"
          >
            {{ $t('nav.users') }}
          </RouterLink>
          <button
            type="button"
            role="menuitem"
            class="block w-full cursor-pointer border-0 bg-transparent px-3 py-1.5 text-left text-ink hover:bg-path"
            @click="onLogout"
          >
            {{ $t('nav.logout') }}
          </button>
        </div>
      </div>
      <RouterLink v-else :to="{ name: 'login' }">{{ $t('auth.signIn') }}</RouterLink>
    </nav>
  </header>
</template>
