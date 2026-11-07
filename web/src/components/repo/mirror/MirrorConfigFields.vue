<script setup lang="ts">
import type { MirrorConflictPolicy, MirrorDirection, Repository } from '@/api/types'
import { formatBytes } from '@/lib/bytes'

defineProps<{
  configured: boolean
  hasPassword: boolean
  hasSSHKey: boolean
  showSSH: boolean
  canEditInsecureTLS: boolean
  current: Repository
  inputCls: string
}>()

const enabled = defineModel<boolean>('enabled', { required: true })
const paused = defineModel<boolean>('paused', { required: true })
const direction = defineModel<MirrorDirection>('direction', { required: true })
const url = defineModel<string>('url', { required: true })
const username = defineModel<string>('username', { required: true })
const password = defineModel<string>('password', { required: true })
const clearPassword = defineModel<boolean>('clearPassword', { required: true })
const sshKey = defineModel<string>('sshKey', { required: true })
const sshPassphrase = defineModel<string>('sshPassphrase', { required: true })
const clearSSHKey = defineModel<boolean>('clearSSHKey', { required: true })
const interval = defineModel<string>('interval', { required: true })
const syncWindow = defineModel<string>('syncWindow', { required: true })
const force = defineModel<boolean>('force', { required: true })
const prune = defineModel<boolean>('prune', { required: true })
const refspecs = defineModel<string>('refspecs', { required: true })
const lfs = defineModel<boolean>('lfs', { required: true })
const partial = defineModel<boolean>('partial', { required: true })
const verifySigned = defineModel<boolean>('verifySigned', { required: true })
const maxMiB = defineModel<string>('maxMiB', { required: true })
const conflictPolicy = defineModel<MirrorConflictPolicy>('conflictPolicy', { required: true })
const insecureTLS = defineModel<boolean>('insecureTLS', { required: true })
const ownerOnly = defineModel<boolean>('ownerOnly', { required: true })
</script>

<template>
  <label v-if="!configured" class="flex items-center gap-2 text-sm">
    <input v-model="enabled" type="checkbox" class="accent-ink">
    {{ $t('settings.mirrorEnable') }}
  </label>

  <template v-if="configured || enabled">
    <slot name="before-fields" />
    <label v-if="configured" class="flex items-center gap-2 text-sm">
      <input v-model="paused" type="checkbox" class="accent-ink">
      {{ $t('mirrorForm.paused') }}
    </label>
    <p v-if="configured" class="-mt-2 text-xs text-ink-muted">{{ $t('mirrorForm.pausedHint') }}</p>

    <label class="block text-sm">
      <span class="mb-1 block font-semibold">{{ $t('settings.mirrorDirection') }}</span>
      <select v-model="direction" :class="inputCls">
        <option value="pull">{{ $t('settings.mirrorPull') }}</option>
        <option value="push">{{ $t('settings.mirrorPush') }}</option>
      </select>
    </label>
    <p class="text-xs text-ink-muted">
      {{ direction === 'push' ? $t('settings.mirrorPushHint') : $t('settings.mirrorPullHint') }}
    </p>

    <label class="block text-sm">
      <span class="mb-1 block font-semibold">{{ $t('settings.mirrorURL') }}</span>
      <input
        v-model="url"
        type="text"
        required
        spellcheck="false"
        autocapitalize="off"
        :class="[inputCls, 'font-mono']"
        :placeholder="$t('modal.mirrorURLPlaceholder')"
      >
    </label>

    <label class="block text-sm">
      <span class="mb-1 block font-semibold">{{ $t('settings.mirrorUsername') }}</span>
      <input v-model="username" :class="inputCls" :placeholder="$t('modal.optional')" autocomplete="off">
    </label>
    <label class="block text-sm">
      <span class="mb-1 block font-semibold">{{ $t('settings.mirrorPassword') }}</span>
      <input
        v-model="password"
        type="password"
        :class="inputCls"
        :placeholder="hasPassword ? $t('settings.mirrorPasswordKeep') : $t('modal.optional')"
        autocomplete="new-password"
      >
    </label>
    <label v-if="hasPassword" class="flex items-center gap-2 text-sm">
      <input v-model="clearPassword" type="checkbox" class="accent-ink">
      {{ $t('settings.mirrorClearPassword') }}
    </label>

    <template v-if="showSSH">
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('mirrorForm.sshKey') }}</span>
        <textarea
          v-model="sshKey"
          rows="5"
          spellcheck="false"
          autocomplete="off"
          :class="[inputCls, 'resize-y font-mono']"
          :placeholder="hasSSHKey ? $t('mirrorForm.sshKeyKeep') : '-----BEGIN OPENSSH PRIVATE KEY-----'"
        />
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('mirrorForm.sshPassphrase') }}</span>
        <input v-model="sshPassphrase" type="password" :class="inputCls" :placeholder="$t('modal.optional')" autocomplete="new-password">
      </label>
      <label v-if="hasSSHKey" class="flex items-center gap-2 text-sm">
        <input v-model="clearSSHKey" type="checkbox" class="accent-ink">
        {{ $t('mirrorForm.clearSshKey') }}
      </label>
      <p class="text-xs text-ink-muted">{{ $t('mirrorForm.sshKeyHint') }}</p>
    </template>

    <div class="grid gap-4 sm:grid-cols-2">
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('mirrorForm.interval') }}</span>
        <input v-model="interval" :class="[inputCls, 'font-mono']" :placeholder="$t('mirrorForm.intervalPlaceholder')" autocomplete="off">
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('mirrorForm.syncWindow') }}</span>
        <input v-model="syncWindow" :class="[inputCls, 'font-mono']" :placeholder="$t('mirrorForm.syncWindowPlaceholder')" autocomplete="off">
      </label>
    </div>
    <p class="-mt-2 text-xs text-ink-muted">{{ $t('mirrorForm.scheduleHint') }}</p>

    <label class="flex items-center gap-2 text-sm">
      <input v-model="force" type="checkbox" class="accent-ink">
      {{ direction === 'push' ? $t('mirrorForm.forcePush') : $t('mirrorForm.forcePull') }}
    </label>
    <label class="flex items-center gap-2 text-sm">
      <input v-model="prune" type="checkbox" class="accent-ink">
      {{ direction === 'push' ? $t('mirrorForm.prunePush') : $t('mirrorForm.prunePull') }}
    </label>
    <p v-if="force || prune" class="-mt-2 text-xs text-ink-muted">{{ $t('mirrorForm.forcePruneHint') }}</p>

    <label v-if="direction === 'pull'" class="block text-sm">
      <span class="mb-1 block font-semibold">{{ $t('mirrorForm.refspecs') }}</span>
      <textarea
        v-model="refspecs"
        rows="3"
        spellcheck="false"
        :class="[inputCls, 'resize-y font-mono']"
        placeholder="+refs/heads/*:refs/heads/*&#10;+refs/tags/*:refs/tags/*"
      />
      <span class="mt-1 block text-xs text-ink-muted">{{ $t('mirrorForm.refspecsHint') }}</span>
    </label>

    <label v-if="direction === 'push'" class="block text-sm">
      <span class="mb-1 block font-semibold">{{ $t('mirrorForm.conflictPolicy') }}</span>
      <select v-model="conflictPolicy" :class="inputCls">
        <option value="force">{{ $t('mirrorForm.conflictForce') }}</option>
        <option value="reject">{{ $t('mirrorForm.conflictReject') }}</option>
      </select>
      <span class="mt-1 block text-xs text-ink-muted">{{ $t('mirrorForm.conflictHint') }}</span>
    </label>

    <label class="flex items-center gap-2 text-sm">
      <input v-model="lfs" type="checkbox" class="accent-ink">
      {{ $t('mirrorForm.lfs') }}
    </label>

    <label class="flex items-center gap-2 text-sm">
      <input v-model="partial" type="checkbox" class="accent-ink">
      {{ $t('mirrorForm.partial') }}
    </label>
    <p class="-mt-2 text-xs text-ink-muted">{{ $t('mirrorForm.partialHint') }}</p>

    <label class="flex items-center gap-2 text-sm">
      <input v-model="verifySigned" type="checkbox" class="accent-ink">
      {{ $t('mirrorForm.verifySigned') }}
    </label>
    <p class="-mt-2 text-xs text-ink-muted">{{ $t('mirrorForm.verifySignedHint') }}</p>

    <label class="block text-sm">
      <span class="mb-1 block font-semibold">{{ $t('mirrorForm.maxMiB') }}</span>
      <input v-model="maxMiB" type="number" min="0" step="any" :class="inputCls" :placeholder="$t('mirrorForm.maxMiBPlaceholder')">
      <span class="mt-1 block text-xs text-ink-muted">
        {{ $t('mirrorForm.maxMiBHint') }}
        <template v-if="current.mirror_max_bytes">({{ formatBytes(current.mirror_max_bytes) }})</template>
      </span>
    </label>

    <template v-if="canEditInsecureTLS">
      <label class="flex items-center gap-2 text-sm">
        <input v-model="insecureTLS" type="checkbox" class="accent-ink">
        {{ $t('mirrorForm.insecureTls') }}
      </label>
      <p v-if="insecureTLS" class="rounded-lg border border-line bg-warn-soft px-2 py-1.5 text-xs text-warn">
        {{ $t('mirrorForm.insecureTlsWarn') }}
      </p>
    </template>

    <label class="flex items-center gap-2 text-sm">
      <input v-model="ownerOnly" type="checkbox" class="accent-ink">
      {{ $t('mirrorForm.ownerOnly') }}
    </label>
  </template>
</template>
