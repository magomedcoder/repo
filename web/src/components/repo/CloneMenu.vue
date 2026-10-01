<script setup lang="ts">
import { ref } from 'vue'
import Icon from '@/components/ui/Icon.vue'

const props = defineProps<{
  httpsUrl: string
  sshUrl: string
}>()

const open = ref(false)
const mode = ref<'https' | 'ssh'>('https')
const copied = ref(false)

const currentUrl = () => (mode.value === 'https' ? props.httpsUrl : props.sshUrl)

function selectUrl(event: FocusEvent) {
  const target = event.target
  if (target instanceof HTMLInputElement) {
    target.select()
  }
}

async function copy() {
  try {
    await navigator.clipboard.writeText(currentUrl())
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 1500)
  } catch {
    copied.value = false
  }
}
</script>

<template>
  <div class="relative">
    <button type="button" class="btn-primary" @click="open = !open">
      <Icon name="code" />
      {{ $t('nav.code') }}
      <Icon name="chevron" />
    </button>
    <div v-if="open" class="absolute right-0 z-20 mt-2 w-96 rounded-md border border-line bg-white p-3 shadow-lg">
      <p class="mb-2 text-xs font-semibold">{{ $t('repo.clone') }}</p>
      <div class="mb-2 inline-flex overflow-hidden rounded-md border border-line text-xs font-medium">
        <button
          type="button"
          class="px-2 py-1"
          :class="mode === 'https' ? 'bg-paper-2 text-ink' : 'text-ink-muted hover:bg-paper'"
          @click="mode = 'https'"
        >
          {{ $t('repo.cloneHttps') }}
        </button>
        <button
          type="button"
          class="border-l border-line px-2 py-1"
          :class="mode === 'ssh' ? 'bg-paper-2 text-ink' : 'text-ink-muted hover:bg-paper'"
          @click="mode = 'ssh'"
        >
          {{ $t('repo.cloneSsh') }}
        </button>
      </div>
      <div class="flex gap-2">
        <input :value="currentUrl()" readonly class="input font-mono text-xs" @focus="selectUrl" />
        <button type="button" class="btn-ghost shrink-0" @click="copy">
          {{ copied ? $t('common.copied') : $t('common.copy') }}
        </button>
      </div>
    </div>
  </div>
</template>
