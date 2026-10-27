import { ref, type Ref } from 'vue'
import type { PageError } from '@/i18n'
import { pageError } from '@/i18n'

export function useAsyncPage() {
  const loading = ref(true)
  const error: Ref<PageError | string> = ref('')

  async function run<T>(
    loader: () => Promise<T>,
    fallbackKey: Parameters<typeof pageError>[1] = 'errors.loadFailed',
  ): Promise<T | null> {
    loading.value = true
    error.value = ''
    try {
      return await loader()
    } catch (err) {
      error.value = pageError(err, fallbackKey)
      return null
    } finally {
      loading.value = false
    }
  }

  return { loading, error, run }
}
