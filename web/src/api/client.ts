import { detectLocale } from '@/i18n'

export class ApiError extends Error {
  status: number
  code?: string

  constructor(status: number, message: string, code?: string) {
    super(message)
    this.status = status
    this.code = code
  }
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.body && !(init.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  const method = (init.method || 'GET').toUpperCase()
  if (method !== 'GET' && method !== 'HEAD' && !headers.has('X-CSRF-Token')) {
    const match = document.cookie.match(/(?:^|; )repo_csrf=([^;]*)/)
    const token = match?.[1]
    if (token) {
      headers.set('X-CSRF-Token', decodeURIComponent(token))
    }
  }
  if (!headers.has('Accept-Language')) {
    headers.set('Accept-Language', detectLocale())
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
    throw new ApiError(res.status, message, code)
  }

  return data as T
}
