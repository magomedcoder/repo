import { onMounted, ref, watch } from 'vue'
import { adminAccessLogsApi } from '@/api'
import type { AccessLog } from '@/api/types'
import { localizeError } from '@/i18n'

export function useAdminAccessLogs() {
  const items = ref<AccessLog[]>([])
  const total = ref(0)
  const action = ref('')
  const ip = ref('')
  const username = ref('')
  const from = ref('')
  const to = ref('')
  const error = ref('')
  const loading = ref(true)
  const pageSize = 50

  function toRFC3339(local: string, endOfDay = false) {
    if (!local) {
      return undefined
    }

    const d = new Date(endOfDay ? `${local}T23:59:59` : `${local}T00:00:00`)
    if (Number.isNaN(d.getTime())) {
      return undefined
    }
    
    return d.toISOString()
  }

  async function load(reset = true) {
    loading.value = true
    error.value = ''
    try {
      const offset = reset ? 0 : items.value.length
      const res = await adminAccessLogsApi.list({
        action: action.value || undefined,
        ip: ip.value.trim() || undefined,
        username: username.value.trim() || undefined,
        from: toRFC3339(from.value),
        to: toRFC3339(to.value, true),
        limit: pageSize,
        offset,
      })
      total.value = res.total
      items.value = reset ? (res.items ?? []) : [...items.value, ...(res.items ?? [])]
    } catch (err) {
      error.value = localizeError(err, 'errors.loadAccessLogs')
    } finally {
      loading.value = false
    }
  }

  let timer: ReturnType<typeof setTimeout> | undefined
  watch([action, ip, username, from, to], () => {
    clearTimeout(timer)
    timer = setTimeout(() => {
      void load(true)
    }, 250)
  })

  onMounted(() => {
    void load(true)
  })

  return { items, total, action, ip, username, from, to, error, loading, load }
}
