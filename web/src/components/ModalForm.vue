<script setup lang="ts">
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

const visible = ref(true)
const dialogRef = ref<HTMLElement | null>(null)
let previousFocus: Element | null = null
let closing = false

const FOCUSABLE = 'button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), a[href], [tabindex]:not([tabindex="-1"])'

function focusables(): HTMLElement[] {
  if (!dialogRef.value) {
    return []
  }

  return Array.from(dialogRef.value.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(
    (el) => !el.hasAttribute('disabled') && el.tabIndex !== -1,
  )
}

function requestClose() {
  if (closing) {
    return
  }

  closing = true
  visible.value = false
}

function onAfterLeave() {
  emit('close')
}

function onKey(ev: KeyboardEvent) {
  if (ev.key === 'Escape') {
    requestClose()
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
  <Transition name="modal" appear @after-leave="onAfterLeave">
    <div
      v-if="visible"
      class="modal-overlay fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-ink/10 p-4 sm:p-6"
      @click.self="requestClose"
    >
      <div
        ref="dialogRef"
        class="modal-panel my-auto flex max-h-[calc(100vh-2rem)] w-full max-w-md flex-col overflow-hidden border border-line bg-paper shadow-none outline-none sm:max-h-[calc(100vh-3rem)]"
        role="dialog"
        aria-modal="true"
        tabindex="-1"
      >
        <div class="flex shrink-0 items-center justify-between border-b border-line px-4 py-3">
          <h2 class="font-medium text-ink">{{ title }}</h2>
          <button
            type="button"
            class="inline-flex cursor-pointer border-0 bg-transparent p-0 text-ink-muted hover:text-ink"
            :aria-label="$t('common.close')"
            @click="requestClose"
          >
            <Icon name="close" />
          </button>
        </div>
        <form class="flex min-h-0 flex-1 flex-col" @submit.prevent="!busy && emit('submit')">
          <div class="min-h-0 flex-1 space-y-3 overflow-y-auto overscroll-contain px-4 py-3">
            <slot />
          </div>
          <div class="flex shrink-0 justify-end gap-2 border-t border-line px-4 py-3">
            <button type="button" class="inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-ink hover:bg-row-hover" @click="requestClose">
              {{ $t('common.cancel') }}
            </button>
            <button type="submit" class="inline-flex items-center justify-center border border-ink bg-ink px-2.5 py-1 text-[10pt] text-paper hover:bg-ink-muted disabled:cursor-not-allowed disabled:opacity-50" :disabled="busy">
              {{ submitLabel || $t('common.ok') }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.modal-enter-active,
.modal-leave-active {
  transition: opacity 160ms ease;
}

.modal-enter-active .modal-panel,
.modal-leave-active .modal-panel {
  transition:
    opacity 160ms ease,
    transform 160ms ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .modal-panel,
.modal-leave-to .modal-panel {
  opacity: 0;
  transform: translateY(0.5rem) scale(0.98);
}

.modal-enter-to .modal-panel,
.modal-leave-from .modal-panel {
  opacity: 1;
  transform: translateY(0) scale(1);
}

@media (prefers-reduced-motion: reduce) {
  .modal-enter-active,
  .modal-leave-active,
  .modal-enter-active .modal-panel,
  .modal-leave-active .modal-panel {
    transition: none;
  }

  .modal-enter-from .modal-panel,
  .modal-leave-to .modal-panel {
    transform: none;
  }
}
</style>
