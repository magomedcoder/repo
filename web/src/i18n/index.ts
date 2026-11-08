import { createI18n } from 'vue-i18n'
import { ApiError } from '@/api/client'
import en from '../../../langs/en/web.json'
import ru from '../../../langs/ru/web.json'

type MessageSchema = typeof en
export type AppLocale = 'en' | 'ru'

export type PageError =
  | { kind: 'unauthorized' }
  | { kind: 'not_found' }
  | { kind: 'message'; text: string }

const STORAGE_KEY = 'repo.locale'
const SUPPORTED: readonly AppLocale[] = ['en', 'ru']

export function normalizeLocale(tag: string): AppLocale | null {
  const lower = tag.trim().toLowerCase().replaceAll('_', '-')
  if (!lower) {
    return null
  }

  if (lower === 'ru' || lower.startsWith('ru-')) {
    return 'ru'
  }

  if (lower === 'en' || lower.startsWith('en-')) {
    return 'en'
  }

  return null
}

export function detectBrowserLocale(): AppLocale {
  const candidates: string[] = []
  if (typeof navigator !== 'undefined') {
    if (Array.isArray(navigator.languages)) {
      candidates.push(...navigator.languages)
    }

    if (navigator.language) {
      candidates.push(navigator.language)
    }
  }

  for (const tag of candidates) {
    const matched = normalizeLocale(tag)
    if (matched) {
      return matched
    }
  }

  return 'en'
}

export function detectLocale(): AppLocale {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved === 'en' || saved === 'ru') {
      return saved
    }
  } catch {}

  return detectBrowserLocale()
}

export function getLocale(): AppLocale {
  const value = i18n.global.locale.value
  return value === 'ru' ? 'ru' : 'en'
}

export const i18n = createI18n<MessageSchema, AppLocale, false>({
  legacy: false,
  locale: detectLocale(),
  fallbackLocale: 'en',
  messages: { en, ru },
})

function applyDocumentLocale(locale: AppLocale) {
  if (typeof document === 'undefined') {
    return
  }

  document.documentElement.lang = locale
  document.documentElement.dir = 'ltr'
}

export function setLocale(locale: AppLocale) {
  if (!SUPPORTED.includes(locale)) {
    locale = 'en'
  }

  i18n.global.locale.value = locale
  try {
    localStorage.setItem(STORAGE_KEY, locale)
  } catch {}

  applyDocumentLocale(locale)
  document.title = i18n.global.t('brand.title')
}

setLocale(detectLocale())

export function localizeError(err: unknown, fallbackKey: Parameters<typeof i18n.global.t>[0]): string {
  if (err instanceof ApiError) {
    if (err.code === 'too_many_attempts' && err.retryAfterSec && err.retryAfterSec > 0) {
      const mins = Math.max(1, Math.ceil(err.retryAfterSec / 60))
      return i18n.global.t('errors.codes.too_many_attempts_wait', { n: mins })
    }

    if (err.code) {
      const codeKey = `errors.codes.${err.code}` as `errors.codes.${string}`
      if (i18n.global.te(codeKey)) {
        return i18n.global.t(codeKey)
      }
    }
    
    return err.message
  }

  return i18n.global.t(fallbackKey)
}

export function pageError(err: unknown, fallbackKey: Parameters<typeof i18n.global.t>[0]): PageError {
  if (err instanceof ApiError && err.status === 401) {
    return { kind: 'unauthorized' }
  }

  if (err instanceof ApiError && err.status === 404 && err.code === 'repository_not_found') {
    return { kind: 'not_found' }
  }

  return { kind: 'message', text: localizeError(err, fallbackKey) }
}

export function formatDate(value: string): string {
  const tag = getLocale() === 'ru' ? 'ru-RU' : 'en-GB'
  return new Date(value).toLocaleString(tag, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function diffStatus(status: string): string {
  const messages = i18n.global.getLocaleMessage(getLocale()) as MessageSchema
  if (status in messages.status) {
    return messages.status[status as keyof MessageSchema['status']]
  }

  return status
}
