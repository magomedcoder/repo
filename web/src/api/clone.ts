export function cloneUrl(fullPath: string) {
  const path = fullPath.replace(/\.git$/, '')
  const base = (import.meta.env.VITE_GIT_BASE as string | undefined)?.replace(/\/$/, '') || window.location.origin
  return `${base}/${path}.git`
}
