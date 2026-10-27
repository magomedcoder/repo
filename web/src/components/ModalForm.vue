<script setup lang="ts">
import { btnGhost, btnPrimary } from '@/lib/classes'
import { nextTick, onMounted, onUnmounted, ref } from 'vue'
import Icon from '@/components/ui/Icon.vue'

defineProps<{
  title: string
  submitLabel?: string
  busy?: boolean
}>()

const emit = defineEmits<{
  close: []
  submit: []
}>()

const dialogRef = ref<HTMLElement | null>(null)
let previousFocus: Element | null = null

const FOCUSABLE = 'button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), a[href], [tabindex]:not([tabindex="-1"])'

function focusables(): HTMLElement[] {
  if (!dialogRef.value) {
    return []
  }

  return Array.from(dialogRef.value.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(
    (el) => !el.hasAttribute('disabled') && el.tabIndex !== -1,
  )
}

function onKey(ev: KeyboardEvent) {
  if (ev.key === 'Escape') {
    emit('close')
    return
  }

  if (ev.key !== 'Tab' || !dialogRef.value) {
    return
  }

  const list = focusables()
  if (!list.length) {
    return
  }

  const first = list[0]!
  const last = list[list.length - 1]!
  const active = document.activeElement
  if (ev.shiftKey) {
    if (active === first || !dialogRef.value.contains(active)) {
      ev.preventDefault()
      last.focus()
    }
  } else if (active === last) {
    ev.preventDefault()
    first.focus()
  }
}

onMounted(async () => {
  previousFocus = document.activeElement
  window.addEventListener('keydown', onKey)
  await nextTick()
  const list = focusables()
  if (list.length) {
    list[0]!.focus()
  } else {
    dialogRef.value?.focus()
  }
})

onUnmounted(() => {
  window.removeEventListener('keydown', onKey)
  if (previousFocus instanceof HTMLElement) {
    previousFocus.focus()
  }
})
</script>

<template>
  <div
    class="fixed inset-0 z-50 flex items-start justify-center bg-ink/10 p-6 pt-[12vh]"
    @click.self="emit('close')"
  >
    <div
      ref="dialogRef"
      class="w-full max-w-md border border-line bg-paper p-4 shadow-none outline-none"
      role="dialog"
      aria-modal="true"
      tabindex="-1"
    >
      <div class="mb-3 flex items-center justify-between border-b border-line pb-2">
        <h2 class="font-medium text-ink">{{ title }}</h2>
        <button
          type="button"
          class="inline-flex cursor-pointer border-0 bg-transparent p-0 text-ink-muted hover:text-ink"
          :aria-label="$t('common.close')"
          @click="emit('close')"
        >
          <Icon name="close" />
        </button>
      </div>
      <form class="space-y-3" @submit.prevent="!busy && emit('submit')">
        <slot />
        <div class="flex justify-end gap-2 pt-1">
          <button type="button" :class="btnGhost" :disabled="busy" @click="emit('close')">
            {{ $t('common.cancel') }}
          </button>
          <button type="submit" :class="btnPrimary" :disabled="busy">
            {{ submitLabel || $t('modal.createFolder') }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
