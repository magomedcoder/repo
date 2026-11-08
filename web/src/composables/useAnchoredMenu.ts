import { nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import type { CSSProperties, Ref } from 'vue'

type Align = 'start' | 'end'

export function useAnchoredMenu(opts: {
  open: Ref<boolean>
  anchor: Ref<HTMLElement | null>
  menu: Ref<HTMLElement | null>
  width?: string
  align?: Align
  gap?: number
}) {
  const style = ref<CSSProperties>({
    position: 'fixed',
    zIndex: 80,
    visibility: 'hidden',
  })

  async function place() {
    if (!opts.open.value) {
      return
    }

    await nextTick()
    const anchor = opts.anchor.value
    const menu = opts.menu.value
    if (!anchor || !menu) {
      return
    }

    const gap = opts.gap ?? 4
    const pad = 8
    const r = anchor.getBoundingClientRect()
    if (opts.width) {
      menu.style.width = opts.width
    }

    const mw = menu.offsetWidth
    const mh = menu.offsetHeight
    const vw = window.innerWidth
    const vh = window.innerHeight

    let top = r.bottom + gap
    if (top + mh > vh - pad && r.top - gap - mh >= pad) {
      top = r.top - gap - mh
    } else if (top + mh > vh - pad) {
      top = Math.max(pad, vh - pad - mh)
    }

    let left = (opts.align ?? 'end') === 'start' ? r.left : r.right - mw
    if (left < pad) {
      left = pad
    }
    
    if (left + mw > vw - pad) {
      left = Math.max(pad, vw - pad - mw)
    }

    style.value = {
      position: 'fixed',
      top: `${Math.round(top)}px`,
      left: `${Math.round(left)}px`,
      zIndex: 80,
      width: opts.width,
      visibility: 'visible',
    }
  }

  function onReposition() {
    if (opts.open.value) {
      void place()
    }
  }

  watch(
    () => opts.open.value,
    (open) => {
      if (open) {
        void place()
      }
    },
  )

  onMounted(() => {
    window.addEventListener('resize', onReposition)
    window.addEventListener('scroll', onReposition, true)
  })

  onUnmounted(() => {
    window.removeEventListener('resize', onReposition)
    window.removeEventListener('scroll', onReposition, true)
  })

  return { style, place }
}
