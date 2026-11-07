const CONTROL_CHARS = /[\u0000-\u001f\u007f]/

export function safeRedirect(raw: unknown, fallback = '/'): string {
  if (typeof raw !== 'string') {
    return fallback
  }

  let path = raw.trim()
  if (!path || CONTROL_CHARS.test(path)) {
    return fallback
  }

  try {
    path = decodeURIComponent(path)
  } catch {
    return fallback
  }

  if (CONTROL_CHARS.test(path)) {
    return fallback
  }

  if (!path.startsWith('/') || path.startsWith('//') || path.includes('\\')) {
    return fallback
  }

  try {
    const origin = typeof window !== 'undefined' && window.location?.origin ? window.location.origin : 'http://localhost'
    const url = new URL(path, origin)
    if (url.origin !== origin) {
      return fallback
    }
    
    if (!url.pathname.startsWith('/') || url.pathname.startsWith('//')) {
      return fallback
    }

    return `${url.pathname}${url.search}${url.hash}`
  } catch {
    return fallback
  }
}
