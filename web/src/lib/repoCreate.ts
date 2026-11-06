import { mirrorRemoteLabel } from '@/lib/mirror'

export function suggestRepoNameFromURL(raw: string): string {
  const label = mirrorRemoteLabel(raw)
  const last = label.split(/[/:]/).filter(Boolean).pop() || ''
  return last.toLowerCase().replace(/[^a-z0-9._-]/g, '-').replace(/^-+|-+$/g, '').slice(0, 100)
}
