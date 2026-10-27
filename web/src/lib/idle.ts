import { i18n } from '@/i18n'

export function idleAge(iso?: string | null): string {
  if (!iso) {
    return ''
  }

  const then = new Date(iso).getTime()
  if (Number.isNaN(then)) {
    return ''
  }

  const t = i18n.global.t
  const minutes = Math.max(0, (Date.now() - then) / 60000)
  if (minutes < 2) {
    return t('index.idleMin', 1)
  }

  if (minutes < 90) {
    return t('index.idleMin', Math.round(minutes))
  }

  const hours = minutes / 60
  if (hours < 36) {
    return t('index.idleHour', Math.round(hours))
  }

  const days = hours / 24
  if (days < 12) {
    return t('index.idleDay', Math.round(days))
  }

  const weeks = days / 7
  if (weeks < 8) {
    return t('index.idleWeek', Math.round(weeks))
  }

  const months = days / 30
  if (months < 20) {
    return t('index.idleMonth', Math.round(months))
  }

  return t('index.idleYear', Math.round(days / 365))
}
