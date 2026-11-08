<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { cloneUrl } from '@/api'
import { useAnchoredMenu } from '@/composables/useAnchoredMenu'
import Icon from '@/components/ui/Icon.vue'

const props = defineProps<{
  repoPath: string
}>()

const open = ref(false)
const root = ref<HTMLElement | null>(null)
const menu = ref<HTMLElement | null>(null)
const copied = ref(false)
let copiedTimer: ReturnType<typeof setTimeout> | undefined

const { style: menuStyle, place: placeMenu } = useAnchoredMenu({
  open,
  anchor: root,
  menu,
  width: 'min(22rem, calc(100vw - 2rem))',
  align: 'end',
})

const url = () => cloneUrl(props.repoPath)

function toggle() {
  open.value = !open.value
  if (!open.value) {
    copied.value = false
  }
}

function close() {
  open.value = false
  copied.value = false
}

async function copy() {
  try {
    await navigator.clipboard.writeText(url())
    copied.value = true
    clearTimeout(copiedTimer)
    copiedTimer = setTimeout(() => {
      copied.value = false
    }, 1600)
  } catch {
    copied.value = false
  }
}

function onDocPointerDown(ev: PointerEvent) {
  if (!(ev.target instanceof Node) || !open.value) {
    return
  }

  const target = ev.target
  if (root.value?.contains(target) || menu.value?.contains(target)) {
    return
  }

  close()
}

function onDocKeydown(ev: KeyboardEvent) {
  if (ev.key === 'Escape') {
    close()
  }
}

onMounted(() => {
  document.addEventListener('pointerdown', onDocPointerDown)
  document.addEventListener('keydown', onDocKeydown)
})

onUnmounted(() => {
  document.removeEventListener('pointerdown', onDocPointerDown)
  document.removeEventListener('keydown', onDocKeydown)
  clearTimeout(copiedTimer)
})
</script>

<template>
  <div ref="root" class="relative">
    <button
      type="button"
      class="btn-primary !h-8 !min-h-8 !gap-1.5 !px-2.5 !py-0 !text-[10pt] !leading-none"
      :aria-expanded="open"
      aria-haspopup="dialog"
      @click="toggle"
    >
      <Icon name="download" size="sm" />
      {{ $t('repo.code') }}
      <Icon name="chevron-down" size="sm" />
    </button>

    <Teleport to="body">
      <Transition name="pop">
        <div
          v-if="open"
          ref="menu"
          role="dialog"
          :aria-label="$t('repo.clone')"
          class="glass-menu p-3"
          :style="menuStyle"
          @vue:mounted="placeMenu"
        >
          <p class="mb-2 text-[11px] font-semibold uppercase tracking-wide text-ink-muted">
            {{ $t('repo.cloneHttps') }}
          </p>
          <div class="flex gap-2">
            <input
              class="field font-mono text-[11px]"
              :value="url()"
              readonly
              @focus="($event.target as HTMLInputElement).select()"
            >
            <button type="button" class="btn shrink-0" @click="copy">
              <Icon :name="copied ? 'check' : 'copy'" size="sm" />
              {{ copied ? $t('repo.copied') : $t('common.copy') }}
            </button>
          </div>
          <p class="mt-2 text-[11px] leading-snug text-ink-muted">
            {{ $t('repo.cloneAuthHint') }}
          </p>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>
