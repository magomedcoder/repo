<script setup lang="ts">
import { tabBar, tabLink, tabLinkOn } from '@/lib/classes'
import { computed, onMounted, provide, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { reposApi } from '@/api'
import type { Repository } from '@/api/types'
import { useAuth } from '@/composables/useAuth'

const props = defineProps<{
  repoPath: string
  tab: 'summary' | 'log' | 'tree' | 'commit' | 'refs' | 'about' | 'issues' | 'pulls' | 'settings' | 'code'
}>()

const auth = useAuth()
const repo = ref<Repository | null>(null)
provide('repo', repo)

const parts = computed(() => props.repoPath.split('/').filter(Boolean))
const folderSlug = computed(() => (parts.value.length > 1 ? parts.value.slice(0, -1).join('/') : ''))
const repoName = computed(() => parts.value[parts.value.length - 1] || props.repoPath)
const isOwner = computed(() => !!auth.user.value?.username && !!repo.value?.owner && auth.user.value.username === repo.value.owner)

function active(name: string) {
  if (name === 'summary') {
    return props.tab === 'summary' || props.tab === 'code'
  }

  if (name === 'tree') {
    return props.tab === 'tree'
  }

  if (name === 'log') {
    return props.tab === 'log' || props.tab === 'commit'
  }

  return props.tab === name
}

async function load() {
  try {
    repo.value = await reposApi.get(props.repoPath)
  } catch {
    repo.value = null
  }
}

onMounted(load)
watch(() => props.repoPath, load)
</script>

<template>
  <div>
    <div class="cgit-path mb-0 font-mono text-[90%]">
      <RouterLink to="/" class="text-ink-muted no-underline hover:underline">{{ $t('nav.root') }}</RouterLink>
      <template v-if="folderSlug">
        <span> / </span>
        <RouterLink
          v-if="repo?.folder_id != null"
          :to="{ name: 'folder', params: { id: String(repo.folder_id) } }"
          class="text-ink-muted no-underline hover:underline"
        >
          {{ folderSlug }}
        </RouterLink>
        <span v-else class="text-ink-muted">{{ folderSlug }}</span>
      </template>
      <span> / </span>
      <RouterLink
        class="font-bold no-underline hover:underline"
        :to="{ name: 'repo', params: { repoPath } }"
      >
        {{ repoName }}
      </RouterLink>
      <span v-if="repo?.description" class="ml-3 font-sans text-ink-muted">{{ repo.description }}</span>
    </div>

    <nav :class="tabBar" :aria-label="$t('nav.issues')">
      <RouterLink
        :to="{ name: 'repo', params: { repoPath } }"
        :class="[tabLink, active('summary') && tabLinkOn]"
      >
        {{ $t('index.summary') }}
      </RouterLink>
      <RouterLink
        :to="{ name: 'repo-commits', params: { repoPath } }"
        :class="[tabLink, active('log') && tabLinkOn]"
      >
        {{ $t('index.log') }}
      </RouterLink>
      <RouterLink
        :to="{ name: 'repo-tree', params: { repoPath } }"
        :class="[tabLink, active('tree') && tabLinkOn]"
      >
        {{ $t('index.tree') }}
      </RouterLink>
      <RouterLink
        :to="{ name: 'repo-refs', params: { repoPath } }"
        :class="[tabLink, active('refs') && tabLinkOn]"
      >
        {{ $t('nav.refs') }}
      </RouterLink>
      <RouterLink
        :to="{ name: 'repo-about', params: { repoPath } }"
        :class="[tabLink, active('about') && tabLinkOn]"
      >
        {{ $t('nav.about') }}
      </RouterLink>
      <RouterLink
        :to="{ name: 'repo-issues', params: { repoPath } }"
        :class="[tabLink, active('issues') && tabLinkOn]"
      >
        {{ $t('nav.issues') }}
      </RouterLink>
      <RouterLink
        :to="{ name: 'repo-pulls', params: { repoPath } }"
        :class="[tabLink, active('pulls') && tabLinkOn]"
      >
        {{ $t('nav.pulls') }}
      </RouterLink>
      <RouterLink
        v-if="isOwner"
        :to="{ name: 'repo-settings', params: { repoPath } }"
        :class="[tabLink, active('settings') && tabLinkOn]"
      >
        {{ $t('nav.settings') }}
      </RouterLink>
    </nav>

    <div class="pt-4">
      <slot />
    </div>
  </div>
</template>
