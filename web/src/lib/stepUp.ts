import { i18n } from '@/i18n'

export function promptStepUpPassword(messageKey = 'common.confirmPassword'): string | null {
  const password = window.prompt(i18n.global.t(messageKey), '')
  if (password == null || password === '') {
    return null
  }
  
  return password
}
