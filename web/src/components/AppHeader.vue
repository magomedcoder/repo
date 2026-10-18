<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'

const headBar = 'flex items-center border-b border-line bg-white px-3 py-2'
const headLink = 'inline-block px-2 text-[13px] leading-none text-ink-muted hover:text-ink hover:no-underline'

const auth = useAuth()
const route = useRoute()
const router = useRouter()
const searchQ = ref(typeof route.query.q === 'string' ? route.query.q : '')

const username = computed(() => auth.user.value?.username || '')

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
  <header>
    <nav :class="headBar" :aria-label="$t('nav.breadcrumb')">
      <RouterLink
        to="/"
        class="mr-2 inline-block text-[15px] font-semibold leading-none text-ink no-underline"
      >Repo</RouterLink>
      <span class="flex-1" />
      <form class="mx-1 flex items-center" @submit.prevent="onSearch">
        <input
          v-model="searchQ"
          type="search"
          class="h-7 w-[14em] rounded-sm border border-line bg-paper-2 px-2 text-[13px] leading-none text-ink placeholder:text-ink-muted"
          :placeholder="$t('search.placeholder')"
          :aria-label="$t('search.title')"
        >
      </form>
      <template v-if="username">
        <RouterLink :class="headLink" :to="{ name: 'profile' }">{{ username }}</RouterLink>
        <RouterLink :class="headLink" :to="{ name: 'tokens' }">{{ $t('nav.accessTokens') }}</RouterLink>
        <button type="button" :class="headLink" @click="onLogout">{{ $t('nav.logout') }}</button>
      </template>
      <template v-else>
        <RouterLink :class="headLink" :to="{ name: 'login' }">{{ $t('auth.signIn') }}</RouterLink>
        <RouterLink :class="headLink" :to="{ name: 'register' }">{{ $t('auth.register') }}</RouterLink>
      </template>
    </nav>
  </header>
</template>
