import { onMounted, ref, watch } from 'vue'
import { adminAccessLogsApi } from '@/api'
import type { AccessLog } from '@/api/types'
import { localizeError } from '@/i18n'

export function useAdminAccessLogs() {
  const items = ref<AccessLog[]>([])
  const total = ref(0)
  const page = ref(1)
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

  async function load() {
    loading.value = true
    error.value = ''
    try {
      const res = await adminAccessLogsApi.list({
        action: action.value || undefined,
        ip: ip.value.trim() || undefined,
        username: username.value.trim() || undefined,
        from: toRFC3339(from.value),
        to: toRFC3339(to.value, true),
        limit: pageSize,
        offset: (page.value - 1) * pageSize,
      })
      total.value = res.total
      items.value = res.items ?? []
      const maxPage = Math.max(1, Math.ceil((res.total || 0) / pageSize) || 1)
      if (page.value > maxPage) {
        page.value = maxPage
        await load()
        return
      }
    } catch (err) {
      error.value = localizeError(err, 'errors.loadAccessLogs')
    } finally {
      loading.value = false
    }
  }

  async function setPage(next: number) {
    page.value = next
    await load()
  }

  let timer: ReturnType<typeof setTimeout> | undefined
  watch([action, ip, username, from, to], () => {
    clearTimeout(timer)
    timer = setTimeout(() => {
      page.value = 1
      void load()
    }, 250)
  })

  onMounted(() => {
    void load()
  })

  return {
    items,
    total,
    page,
    pageSize,
    action,
    ip,
    username,
    from,
    to,
    error,
    loading,
    load,
    setPage,
  }
}
