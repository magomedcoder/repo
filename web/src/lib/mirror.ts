import type { Repository } from '@/api/types'

export type MirrorUiStatus = 'syncing' | 'ok' | 'error' | 'paused' | 'idle' | 'skipped'

type MirrorStateFields = Pick<Repository, 'mirror_direction' | 'mirror_enabled' | 'mirror_status' | 'mirror_last_error' | 'mirror_last_sync_at'>

export function hasMirror(repo: Pick<Repository, 'mirror_direction'> | null | undefined): boolean {
  const dir = repo?.mirror_direction
  return dir === 'pull' || dir === 'push'
}

export function mirrorUiStatus(repo: MirrorStateFields | null | undefined): MirrorUiStatus | '' {
  if (!repo || !hasMirror(repo)) {
    return ''
  }

  if (repo.mirror_enabled === false || repo.mirror_status === 'paused') {
    return 'paused'
  }

  switch (repo.mirror_status) {
    case 'syncing':
    case 'ok':
    case 'error':
    case 'idle':
    case 'skipped':
      return repo.mirror_status
  }

  if (repo.mirror_last_error) {
    return 'error'
  }

  return repo.mirror_last_sync_at ? 'ok' : 'idle'
}

export function mirrorRemoteLabel(raw: string | undefined): string {
  const url = (raw || '').trim()
  if (!url) {
    return ''
  }

  try {
    const u = new URL(url)
    const path = u.pathname.replace(/\.git$/, '').replace(/^\/+|\/+$/g, '')
    return path ? `${u.host}/${path}` : u.host
  } catch {
    return url.replace(/^(https?|ssh|git):\/\//, '').replace(/^[^@/]+@/, '').replace(/\.git$/, '')
  }
}

export function mirrorWebURL(raw: string | undefined): string {
  const url = (raw || '').trim()
  if (!url) {
    return ''
  }

  const scp = url.match(/^[^@/\s]+@([^:/\s]+):(.+?)(?:\.git)?\/?$/)
  if (scp) {
    return `https://${scp[1]}/${scp[2]}`
  }

  try {
    const u = new URL(url)
    if (u.protocol === 'http:' || u.protocol === 'https:') {
      return `${u.protocol}//${u.host}${u.pathname.replace(/\.git$/, '').replace(/\/+$/, '')}`
    }

    if (u.protocol === 'ssh:' || u.protocol === 'git:') {
      return `https://${u.hostname}${u.pathname.replace(/\.git$/, '').replace(/\/+$/, '')}`
    }
  } catch {}

  return ''
}

export function refspecsToText(value: string[] | string | undefined): string {
  if (!value) {
    return ''
  }

  return Array.isArray(value) ? value.join('\n') : value
}

export function normalizeRefspecs(text: string): string {
  return text.split(/\r?\n/).map((line) => line.trim()).filter(Boolean).join('\n')
}

export function isSSHURL(raw: string): boolean {
  const url = raw.trim()
  return /^ssh:\/\//i.test(url) || /^[^@/\s]+@[^:/\s]+:/.test(url)
}
