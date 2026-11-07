<script setup lang="ts">
import { computed, inject, onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { usersApi } from '@/api'
import logoUrl from '@/assets/logo.svg'
import Icon from '@/components/ui/Icon.vue'
import { useAuth } from '@/composables/useAuth'
import { useSite } from '@/composables/useSite'

const auth = useAuth()
const { name: siteName } = useSite()
const route = useRoute()
const router = useRouter()
const searchQ = ref(typeof route.query.q === 'string' ? route.query.q : '')

const openNewRepo = inject<() => void>('openNewRepo', () => {})
const openImportRepo = inject<(url?: string) => void>('openImportRepo', () => {})
const openMirrorRepo = inject<(url?: string) => void>('openMirrorRepo', () => {})

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

function onNewRepo() {
  closeMenus()
  openNewRepo()
}

function onImportRepo() {
  closeMenus()
  openImportRepo()
}

function onMirrorRepo() {
  closeMenus()
  openMirrorRepo()
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
  <header class="glass sticky top-0 z-40 flex items-center gap-2 px-2.5 py-2 sm:gap-3 sm:px-4 sm:py-2.5">
    <div class="flex min-w-0 shrink-0 items-center gap-2 sm:gap-2.5">
      <RouterLink
        to="/"
        class="inline-flex min-w-0 items-center gap-2 text-[1.2rem] font-medium leading-none tracking-tight text-ink no-underline hover:no-underline sm:gap-2.5 sm:text-[1.35rem]"
      >
        <img
          :src="logoUrl"
          alt=""
          width="28"
          height="28"
          class="size-6 shrink-0 sm:size-7"
        >
        <span class="min-w-0 truncate">{{ siteName }}</span>
      </RouterLink>
    </div>

    <nav class="ml-auto flex shrink-0 items-center justify-end gap-1.5 text-[90%] sm:gap-2">
      <form class="min-w-0" @submit.prevent="onSearch">
        <label class="sr-only" for="global-search">{{ $t('search.title') }}</label>
        <div class="search-field h-8 w-28 sm:w-40">
          <Icon name="search" size="sm" class="shrink-0" />
          <input
            id="global-search"
            v-model="searchQ"
            type="search"
            class="w-full min-w-0 text-[90%]"
            :placeholder="$t('search.title')"
            :aria-label="$t('search.title')"
          >
        </div>
      </form>
      <div v-if="username" ref="createRoot" class="relative z-50">
        <button
          type="button"
          class="icon-btn"
          :aria-expanded="createOpen"
          aria-haspopup="menu"
          :aria-label="$t('nav.create')"
          @click="toggleCreate"
        >
          <Icon name="plus" />
        </button>
        <Transition name="pop">
          <div
            v-if="createOpen"
            role="menu"
            class="glass-menu absolute right-0 top-full z-[70] mt-1 min-w-48 overflow-hidden py-1"
          >
            <button type="button" role="menuitem" class="menu-item" @click="onNewRepo">
              <Icon name="repo" size="sm" class="text-ink-muted" />
              {{ $t('home.newRepository') }}
            </button>
            <button type="button" role="menuitem" class="menu-item" @click="onImportRepo">
              <Icon name="download" size="sm" class="text-ink-muted" />
              {{ $t('home.importRepository') }}
            </button>
            <button type="button" role="menuitem" class="menu-item" @click="onMirrorRepo">
              <Icon name="mirror" size="sm" class="text-ink-muted" />
              {{ $t('home.mirrorRepository') }}
            </button>
          </div>
        </Transition>
      </div>
      <div v-if="username" ref="menuRoot" class="relative z-50">
        <button
          type="button"
          class="flex h-8 w-8 cursor-pointer items-center justify-center overflow-hidden rounded-full border border-line bg-white/55 p-0 text-[11px] font-semibold text-ink transition hover:-translate-y-px hover:bg-white/75"
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
        <Transition name="pop">
          <div
            v-if="menuOpen"
            role="menu"
            class="glass-menu absolute right-0 top-full z-[70] mt-1 min-w-48 overflow-hidden py-1"
          >
            <RouterLink
              role="menuitem"
              class="menu-item"
              :to="{ name: 'profile' }"
              @click="closeMenus"
            >
              <Icon name="user" size="sm" class="text-ink-muted" />
              {{ $t('nav.profile') }}
            </RouterLink>
            <RouterLink
              role="menuitem"
              class="menu-item"
              :to="{ name: 'teams' }"
              @click="closeMenus"
            >
              <Icon name="users" size="sm" class="text-ink-muted" />
              {{ $t('nav.teams') }}
            </RouterLink>
            <RouterLink
              v-if="isAdmin"
              role="menuitem"
              class="menu-item"
              :to="{ name: 'cp' }"
              @click="closeMenus"
            >
              <Icon name="gear" size="sm" class="text-ink-muted" />
              {{ $t('nav.controlPanel') }}
            </RouterLink>
            <button type="button" role="menuitem" class="menu-item" @click="onLogout">
              <Icon name="logout" size="sm" class="text-ink-muted" />
              {{ $t('nav.logout') }}
            </button>
          </div>
        </Transition>
      </div>
      <RouterLink
        v-else
        class="btn"
        :to="{ name: 'login' }"
      >
        <Icon name="user" size="sm" />
        {{ $t('auth.signIn') }}
      </RouterLink>
    </nav>
  </header>
</template>
