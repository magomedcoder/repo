export function safeRedirect(raw: unknown, fallback = '/'): string {
  if (typeof raw !== 'string') {
    return fallback
  }

  let path = raw.trim()
  try {
    path = decodeURIComponent(path)
  } catch {
    return fallback
  }

  if (
    !path.startsWith('/') ||
    path.startsWith('//') ||
    path.includes('//') ||
    path.includes('\\') ||
    !/^\/[^/]/.test(path)
  ) {
    return fallback
  }

  return path
}
