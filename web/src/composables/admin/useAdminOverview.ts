import { onMounted, ref } from 'vue'
import { adminOverviewApi, type AdminOverview } from '@/api'
import { localizeError } from '@/i18n'

export function useAdminOverview() {
  const data = ref<AdminOverview | null>(null)
  const loading = ref(true)
  const error = ref('')

  async function load() {
    loading.value = true
    error.value = ''
    try {
      data.value = await adminOverviewApi.get()
    } catch (err) {
      error.value = localizeError(err, 'errors.loadFailed')
    } finally {
      loading.value = false
    }
  }

  onMounted(() => {
    void load()
  })

  return { data, loading, error, load }
}
