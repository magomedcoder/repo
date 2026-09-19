<script setup lang="ts">
import { ref } from 'vue'
import Icon from '@/components/ui/Icon.vue'

const props = defineProps<{
  url: string
}>()

function selectUrl(event: FocusEvent) {
  const target = event.target
  if (target instanceof HTMLInputElement) {
    target.select()
  }
}
const open = ref(false)
const copied = ref(false)

async function copy() {
  try {
    await navigator.clipboard.writeText(props.url)
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
    <button
      type="button"
      class="btn-primary"
      @click="open = !open"
    >
      <Icon name="code" />
      {{ $t('nav.code') }}
      <Icon name="chevron" />
    </button>
    <div v-if="open" class="absolute right-0 z-20 mt-2 w-80 rounded-md border border-line bg-white p-3 shadow-lg">
      <p class="mb-2 text-xs font-semibold">{{ $t('repo.clone') }}</p>
      <div class="flex gap-2">
        <input
          :value="url"
          readonly
          class="input font-mono text-xs"
          @focus="selectUrl"
        />
        <button
          type="button"
          class="btn-ghost shrink-0"
          @click="copy"
        >{{ copied ? $t('common.copied') : $t('common.copy') }}</button>
      </div>
    </div>
  </div>
</template>
