import { createI18n } from 'vue-i18n'
import { ApiError } from '@/api/client'
import en from '../../../langs/en/web.json'
import ru from '../../../langs/ru/web.json'

export type MessageSchema = typeof en
export type AppLocale = 'en' | 'ru'

const STORAGE_KEY = 'repo.locale'

export function detectLocale(): AppLocale {
  const saved = localStorage.getItem(STORAGE_KEY)
  if (saved === 'en' || saved === 'ru') {
    return saved
  }

  return navigator.language.toLowerCase().startsWith('ru') ? 'ru' : 'en'
}

export const i18n = createI18n<MessageSchema, AppLocale, false>({
  legacy: false,
  locale: detectLocale(),
  fallbackLocale: 'en',
  messages: { en, ru },
})

export function setLocale(locale: AppLocale) {
  i18n.global.locale.value = locale
  localStorage.setItem(STORAGE_KEY, locale)
  document.documentElement.lang = locale
}

setLocale(detectLocale())

export function localizeError(err: unknown, fallbackKey: Parameters<typeof i18n.global.t>[0]): string {
  if (err instanceof ApiError) {
    return err.message
  }

  return i18n.global.t(fallbackKey)
}

export function formatDate(value: string): string {
  const tag = i18n.global.locale.value === 'ru' ? 'ru-RU' : 'en-GB'
  return new Date(value).toLocaleString(tag)
}

export function diffStatus(status: string): string {
  const messages = i18n.global.getLocaleMessage(i18n.global.locale.value) as MessageSchema
  if (status in messages.status) {
    return messages.status[status as keyof MessageSchema['status']]
  }

  return status
}
