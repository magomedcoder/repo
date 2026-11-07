<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { tokensApi } from '@/api'
import type { AccessToken } from '@/api/types'
import PageState from '@/components/ui/PageState.vue'
import { formatDate, i18n, localizeError } from '@/i18n'
import Icon from '@/components/ui/Icon.vue'
import PageHeader from '@/components/ui/PageHeader.vue'
import Surface from '@/components/ui/Surface.vue'
import SurfaceHeader from '@/components/ui/SurfaceHeader.vue'
import EmptyState from '@/components/ui/EmptyState.vue'

const tokens = ref<AccessToken[]>([])
const name = ref('')
const expiresInDays = ref<number | null>(90)
const scope = ref<'git:write' | 'git:read'>('git:write')
const error = ref('')
const loading = ref(true)
const creating = ref(false)
const showForm = ref(false)
const createdToken = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await tokensApi.list()
    tokens.value = res.tokens ?? []
  } catch (err) {
    error.value = localizeError(err, 'errors.loadTokens')
  } finally {
    loading.value = false
  }
}

async function create() {
  creating.value = true
  error.value = ''
  createdToken.value = ''
  try {
    const body: { name: string; scope: string; expires_in?: number } = {
      name: name.value.trim(),
      scope: scope.value,
    }

    if (expiresInDays.value != null && expiresInDays.value > 0) {
      body.expires_in = expiresInDays.value
    }

    const res = await tokensApi.create(body)
    createdToken.value = res.token
    name.value = ''
    expiresInDays.value = 90
    scope.value = 'git:write'
    showForm.value = false
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.createToken')
  } finally {
    creating.value = false
  }
}

async function revoke(id: number) {
  if (!confirm(i18n.global.t('tokens.confirmRevoke'))) {
    return
  }

  error.value = ''
  try {
    await tokensApi.revoke(id)
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.deleteFailed')
  }
}

async function copyToken() {
  if (!createdToken.value) {
    return
  }
  try {
    await navigator.clipboard.writeText(createdToken.value)
  } catch {}
}

onMounted(load)
</script>

<template>
  <div>
    <PageHeader :title="$t('tokens.title')" icon="key">
      <template #actions>
        <button type="button" class="btn-primary" @click="showForm = !showForm">
          <Icon name="plus" size="sm" />
          {{ $t('tokens.add') }}
        </button>
      </template>
    </PageHeader>

    <Surface
      v-if="createdToken"
      class="mb-4"
    >
      <SurfaceHeader>{{ $t('tokens.createdTitle') }}</SurfaceHeader>
      <div class="p-3 sm:p-4">
        <p class="text-xs text-ink-muted">{{ $t('tokens.createdHint') }}</p>
        <pre class="glass mt-2 overflow-x-auto p-3 font-mono text-xs">{{ createdToken }}</pre>
        <div class="mt-3 flex flex-wrap gap-2">
          <button type="button" class="btn-primary" @click="copyToken"><Icon name="copy" size="sm" />{{ $t('common.copy') }}</button>
          <button type="button" class="btn" @click="createdToken = ''"><Icon name="close" size="sm" />{{ $t('common.close') }}</button>
        </div>
      </div>
    </Surface>

    <Surface
      v-if="showForm"
      as="form"
      class="mb-4"
      @submit.prevent="create"
    >
      <SurfaceHeader>{{ $t('tokens.add') }}</SurfaceHeader>
      <div class="p-3 sm:p-4 space-y-3">
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('tokens.nameField') }}</span>
          <input
            v-model="name"
            class="field"
            required
            maxlength="64"
            :placeholder="$t('tokens.namePlaceholder')"
          />
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('tokens.scope') }}</span>
          <select v-model="scope" class="field">
            <option value="git:write">{{ $t('tokens.scopeWrite') }}</option>
            <option value="git:read">{{ $t('tokens.scopeRead') }}</option>
          </select>
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('tokens.expiresIn') }}</span>
          <input
            v-model.number="expiresInDays"
            type="number"
            min="1"
            max="3650"
            class="field"
            :placeholder="$t('tokens.expiresInPlaceholder')"
          />
        </label>
        <p class="text-xs text-ink-muted">{{ $t('tokens.authHint') }}</p>
        <button type="submit" class="btn-primary" :disabled="creating">
          {{ creating ? $t('tokens.creating') : $t('tokens.submit') }}
        </button>
      </div>
    </Surface>

    <p v-if="error" class="mb-3 text-sm text-warn">{{ error }}</p>
    <PageState :loading="loading">
      <Surface as="ul" class="divide-y divide-line">
        <li
          v-for="token in tokens"
          :key="token.id"
          class="flex flex-wrap items-start justify-between gap-3 px-4 py-3"
        >
          <div class="min-w-0">
            <p class="font-semibold">{{ token.name }}</p>
            <p class="mt-1 font-mono text-xs text-ink-muted">
              {{ $t('tokens.prefix') }}: {{ token.prefix }}...
            </p>
            <p class="mt-1 text-xs text-ink-muted">{{ token.scope || 'git:write' }}</p>
            <p class="mt-1 text-xs text-ink-muted">{{ formatDate(token.created_at) }}</p>
            <p v-if="token.expires_at" class="mt-0.5 text-xs text-ink-muted">
              {{ $t('tokens.expiresAt') }}: {{ formatDate(token.expires_at) }}
            </p>
            <p v-if="token.last_used_at" class="mt-0.5 text-xs text-ink-muted">
              {{ $t('tokens.lastUsed') }}: {{ formatDate(token.last_used_at) }}
            </p>
          </div>
          <button type="button" class="btn-warn" @click="revoke(token.id)">
            {{ $t('tokens.revoke') }}
          </button>
        </li>
        <EmptyState
          v-if="!tokens.length"
          as="li"
          :message="$t('tokens.empty')"
        />
      </Surface>
    </PageState>
  </div>
</template>
