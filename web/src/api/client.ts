import { getLocale } from '@/i18n'

export class ApiError extends Error {
  status: number
  code?: string
  retryAfterSec?: number

  constructor(status: number, message: string, code?: string, retryAfterSec?: number) {
    super(message)
    this.status = status
    this.code = code
    this.retryAfterSec = retryAfterSec
  }
}

let csrfToken = ''
let csrfPromise: Promise<string> | null = null

async function fetchCSRFToken(): Promise<string> {
  if (csrfToken) {
    return csrfToken
  }

  if (!csrfPromise) {
    csrfPromise = fetch('/api/auth/csrf', { credentials: 'include' })
      .then(async (res) => {
        if (!res.ok) {
          return ''
        }

        const data = (await res.json()) as { csrf_token?: string }
        csrfToken = data.csrf_token || ''
        return csrfToken
      })
      .finally(() => {
        csrfPromise = null
      })
  }

  return csrfPromise
}

export function clearCSRFToken() {
  csrfToken = ''
}

export async function api<T>(path: string, init: RequestInit = {}, retried = false): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.body && !(init.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  const method = (init.method || 'GET').toUpperCase()
  if (method !== 'GET' && method !== 'HEAD' && !headers.has('X-CSRF-Token')) {
    const token = await fetchCSRFToken()
    if (token) {
      headers.set('X-CSRF-Token', token)
    }
  }
  if (!headers.has('Accept-Language')) {
    headers.set('Accept-Language', getLocale())
  }

  const res = await fetch(path, {
    ...init,
    headers,
    credentials: 'include',
  })

  if (res.status === 204) {
    return undefined as T
  }

  const text = await res.text()
  let data: unknown = null
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      data = { error: text }
    }
  }

  if (!res.ok) {
    let message = res.statusText || 'request failed'
    let code: string | undefined
    if (data && typeof data === 'object') {
      const body = data as Record<string, unknown>
      if (typeof body.error === 'string') {
        message = body.error
      }

      if (typeof body.code === 'string') {
        code = body.code
      }
    }

    if (code === 'csrf') {
      clearCSRFToken()
      if (!retried) {
        const retryHeaders = new Headers(init.headers)
        retryHeaders.delete('X-CSRF-Token')
        return api<T>(path, { 
          ...init, 
          headers: retryHeaders
        }, true)
      }
    }

    let retryAfterSec: number | undefined
    const ra = res.headers.get('Retry-After')
    if (ra) {
      const n = Number.parseInt(ra, 10)
      if (Number.isFinite(n) && n > 0) {
        retryAfterSec = n
      }
    }

    throw new ApiError(res.status, message, code, retryAfterSec)
  }

  return data as T
}
