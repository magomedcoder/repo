<script setup lang="ts">
import { tabBar, tabLink, tabLinkOn } from '@/lib/classes'
import { onMounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { reposApi } from '@/api'
import type { Repository } from '@/api/types'

const props = defineProps<{
  owner: string
  repoPath: string
  tab: 'summary' | 'log' | 'tree' | 'commit' | 'refs' | 'about' | 'issues' | 'pulls' | 'settings' | 'code'
}>()

const repo = ref<Repository | null>(null)

const parts = () => props.repoPath.split('/').filter(Boolean)

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
    repo.value = await reposApi.get(props.owner, props.repoPath)
  } catch {
    repo.value = null
  }
}

onMounted(load)
watch(() => [props.owner, props.repoPath], load)
</script>

<template>
  <div>
    <div class="mb-1 px-2 font-mono text-[10pt]">
      <template v-for="(part, i) in parts()" :key="`${part}-${i}`">
        <span v-if="i > 0" class="text-ink"> / </span>
        <RouterLink
          v-if="i === parts().length - 1"
          :to="{ name: 'repo', params: { repoPath } }"
        >
          {{ part }}
        </RouterLink>
            <RouterLink
              v-else
              :to="{
                name: 'repo-tree',
                params: { repoPath },
                query: { path: parts().slice(0, i + 1).join('/') },
              }"
            >
              {{ part }}
            </RouterLink>
      </template>
      <span v-if="repo?.description" class="ml-2 text-ink-muted">{{ repo.description }}</span>
    </div>
    <nav :class="tabBar" :aria-label="$t('nav.breadcrumb')">
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
        :to="{ name: 'repo-settings', params: { repoPath } }"
        :class="[tabLink, active('settings') && tabLinkOn]"
      >
        {{ $t('nav.settings') }}
      </RouterLink>
    </nav>
    <div class="px-1 py-2">
      <slot />
    </div>
  </div>
</template>
