import { ref } from 'vue'
import type { RouteLocationRaw } from 'vue-router'

export interface Crumb {
  label: string
  to?: RouteLocationRaw
  labelKey?: string
}

const crumbs = ref<Crumb[]>([])

export function useBreadcrumbs() {
  function setBreadcrumbs(next: Crumb[]) {
    crumbs.value = next
  }

  function clearBreadcrumbs() {
    crumbs.value = []
  }

  return {
    crumbs,
    setBreadcrumbs,
    clearBreadcrumbs,
  }
}
