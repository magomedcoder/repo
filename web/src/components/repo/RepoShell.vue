<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { reposApi } from '@/api'
import type { Repository } from '@/api/types'
import Icon from '@/components/ui/Icon.vue'
import VisibilityBadge from '@/components/ui/VisibilityBadge.vue'

const props = defineProps<{
  owner: string
  repoPath: string
  tab: 'code' | 'issues' | 'settings'
}>()

const repo = ref<Repository | null>(null)

const parts = () => props.repoPath.split('/').filter(Boolean)

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
    <div class="border-b border-line bg-white">
      <div class="mx-auto max-w-7xl px-4 pt-4">
        <div class="flex flex-wrap items-center gap-2 text-xl leading-tight">
          <span class="flex h-8 w-8 items-center justify-center rounded-full bg-accent-soft text-sm font-semibold text-accent">
            {{ owner.slice(0, 1).toUpperCase() }}
          </span>
          <RouterLink to="/" class="text-accent hover:underline">{{ owner }}</RouterLink>
          <template
            v-for="(part, i) in parts()"
            :key="`${part}-${i}`"
          >
            <span class="text-ink-muted">/</span>
            <RouterLink
              v-if="i === parts().length - 1"
              :to="{
                name: 'repo',
                params: { owner, repoPath }
              }"
              class="font-semibold text-accent hover:underline"
            >
              {{ part }}
            </RouterLink>
            <span v-else>{{ part }}</span>
          </template>
          <VisibilityBadge v-if="repo" :is-private="repo.is_private" />
        </div>
        <p v-if="repo?.description" class="mt-2 text-ink-muted">{{ repo.description }}</p>

        <nav class="mt-4 flex gap-1 overflow-x-auto text-sm" :aria-label="$t('nav.breadcrumb')">
          <RouterLink
            :to="{
              name: 'repo',
              params: { owner, repoPath }
            }"
            class="inline-flex items-center gap-2 border-b-2 px-3 py-2"
            :class="tab === 'code' ? 'border-[#fd8c73] font-semibold' : 'border-transparent text-ink hover:border-line'"
          >
            <Icon name="code" />
            {{ $t('nav.code') }}
          </RouterLink>
          <RouterLink
            :to="{
              name: 'repo-issues',
              params: { owner, repoPath }
            }"
            class="inline-flex items-center gap-2 border-b-2 px-3 py-2"
            :class="tab === 'issues' ? 'border-[#fd8c73] font-semibold' : 'border-transparent text-ink hover:border-line'"
          >
            <Icon name="issue" />
            {{ $t('nav.issues') }}
          </RouterLink>
          <RouterLink
            :to="{
              name: 'repo-settings',
              params: { owner, repoPath }
            }"
            class="inline-flex items-center gap-2 border-b-2 px-3 py-2"
            :class="tab === 'settings' ? 'border-[#fd8c73] font-semibold' : 'border-transparent text-ink hover:border-line'"
          >
            <Icon name="gear" />
            {{ $t('nav.settings') }}
          </RouterLink>
        </nav>
      </div>
    </div>
    <div class="mx-auto max-w-7xl px-4 py-4">
      <slot />
    </div>
  </div>
</template>
