import type { MirrorConflictPolicy, MirrorDirection } from './types'

export interface MirrorConfigureBody {
  enabled?: boolean
  clear?: boolean
  direction?: MirrorDirection | string
  url?: string
  username?: string
  password?: string
  clear_password?: boolean
  ssh_private_key?: string
  ssh_passphrase?: string
  clear_ssh_key?: boolean
  interval?: string
  sync_window?: string
  force?: boolean
  prune?: boolean
  refspecs?: string
  lfs?: boolean
  max_bytes?: number
  conflict_policy?: MirrorConflictPolicy | string
  insecure_tls?: boolean
  owner_only?: boolean
  partial?: boolean
  verify_signed?: boolean
}
