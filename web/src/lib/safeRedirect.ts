export function safeRedirect(raw: unknown, fallback = '/'): string {
  if (typeof raw !== 'string') {
    return fallback
  }
  
  const path = raw.trim()
  if (!path.startsWith('/') || path.startsWith('//') || path.includes('\\')) {
    return fallback
  }

  return path
}
