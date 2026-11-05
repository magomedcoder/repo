import { computed, ref, watch } from 'vue'
import { siteApi } from '@/api'
import { i18n } from '@/i18n'

const name = ref('')
const ready = ref(false)
let loading: Promise<void> | null = null

function applyTitle() {
  document.title = name.value || i18n.global.t('brand.title')
}

watch(() => i18n.global.locale.value, applyTitle)

export function useSite() {
  async function fetchSite(): Promise<void> {
    if (ready.value) {
      return
    }

    if (loading) {
      return loading
    }

    loading = (async () => {
      try {
        const res = await siteApi.get()
        name.value = (res.name || '').trim() || i18n.global.t('brand.name')
      } catch {
        name.value = i18n.global.t('brand.name')
      } finally {
        applyTitle()
        ready.value = true
        loading = null
      }
    })()

    return loading
  }

  return {
    name: computed(() => name.value || i18n.global.t('brand.name')),
    ready,
    fetchSite,
  }
}
