import { getLocale } from '@/i18n'

type Unit = Intl.RelativeTimeFormatUnit

const DIVISIONS: { amount: number; unit: Unit }[] = [
  { amount: 60, unit: 'second' },
  { amount: 60, unit: 'minute' },
  { amount: 24, unit: 'hour' },
  { amount: 7, unit: 'day' },
  { amount: 4.34524, unit: 'week' },
  { amount: 12, unit: 'month' },
  { amount: Number.POSITIVE_INFINITY, unit: 'year' },
]

function relativeLocale(): string {
  return getLocale() === 'ru' ? 'ru-RU' : 'en'
}

export function idleAge(iso?: string | null): string {
  if (!iso) {
    return ''
  }

  const then = new Date(iso).getTime()
  if (Number.isNaN(then)) {
    return ''
  }

  let seconds = Math.round((then - Date.now()) / 1000)
  if (seconds > 0) {
    seconds = 0
  }

  const rtf = new Intl.RelativeTimeFormat(relativeLocale(), {
    numeric: 'auto',
    style: 'long',
  })

  let duration = seconds
  for (const { amount, unit } of DIVISIONS) {
    if (Math.abs(duration) < amount) {
      return rtf.format(Math.round(duration), unit)
    }

    duration /= amount
  }

  return rtf.format(Math.round(duration), 'year')
}
