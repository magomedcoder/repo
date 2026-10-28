import { i18n } from '@/i18n'

const UNITS = ['B', 'KB', 'MB', 'GB', 'TB'] as const

export function formatBytes(bytes: number): string {
  let n = Number(bytes)
  if (!Number.isFinite(n) || n < 0) {
    n = 0
  }

  n = Math.floor(n)
  if (n < 1024) {
    return i18n.global.t('common.sizeB', { n })
  }

  let unit = 0
  let value = n
  while (value >= 1024 && unit < UNITS.length - 1) {
    value /= 1024
    unit++
  }

  const digits = value >= 100 ? 0 : value >= 10 ? 1 : 2
  const tag = i18n.global.locale.value === 'ru' ? 'ru-RU' : 'en-GB'
  const formatted = value.toLocaleString(tag, {
    minimumFractionDigits: 0,
    maximumFractionDigits: digits,
  })

  return i18n.global.t(`common.size${UNITS[unit]}`, { n: formatted })
}
