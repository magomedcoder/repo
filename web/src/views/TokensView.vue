<script setup lang="ts">
import { btnDanger, btnGhost, btnPrimary, field } from '@/lib/classes'
import { onMounted, onUnmounted, ref } from 'vue'
import { tokensApi } from '@/api'
import type { AccessToken } from '@/api/types'
import PageState from '@/components/ui/PageState.vue'
import { useBreadcrumbs } from '@/composables/useBreadcrumbs'
import { formatDate, i18n, localizeError } from '@/i18n'

const { setBreadcrumbs, clearBreadcrumbs } = useBreadcrumbs()

const tokens = ref<AccessToken[]>([])
const name = ref('')
const error = ref('')
const loading = ref(true)
const creating = ref(false)
const showForm = ref(false)
const createdToken = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    setBreadcrumbs([
      { 
        label: 'profile', 
        labelKey: 'nav.profile'
      },
      { 
        label: 'tokens', 
        labelKey: 'nav.accessTokens' 
      },
    ])
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
    const res = await tokensApi.create({ 
      name: name.value.trim() 
    })
    createdToken.value = res.token
    name.value = ''
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
onUnmounted(clearBreadcrumbs)
</script>

<template>
  <div class="mx-auto max-w-3xl px-4 py-6">
    <div class="mb-4 flex flex-wrap items-start justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold">{{ $t('tokens.title') }}</h1>
        <p class="mt-1 text-sm text-ink-muted">{{ $t('tokens.lead') }}</p>
      </div>
      <button type="button" :class="btnPrimary" @click="showForm = !showForm">
        {{ $t('tokens.add') }}
      </button>
    </div>

    <div
      v-if="createdToken"
      class="mb-4 rounded-md border border-moss/40 bg-paper p-4"
    >
      <p class="text-sm font-semibold">{{ $t('tokens.createdTitle') }}</p>
      <p class="mt-1 text-xs text-ink-muted">{{ $t('tokens.createdHint') }}</p>
      <pre class="mt-2 overflow-x-auto rounded bg-white p-3 font-mono text-xs">{{ createdToken }}</pre>
      <div class="mt-3 flex flex-wrap gap-2">
        <button type="button" :class="btnPrimary" @click="copyToken">{{ $t('common.copy') }}</button>
        <button type="button" :class="btnGhost" @click="createdToken = ''">{{ $t('common.close') }}</button>
      </div>
    </div>

    <form
      v-if="showForm"
      class="mb-4 space-y-3 rounded-md border border-line bg-white p-4"
      @submit.prevent="create"
    >
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('tokens.nameField') }}</span>
        <input
          v-model="name"
          :class="field"
          required
          maxlength="64"
          :placeholder="$t('tokens.namePlaceholder')"
        />
      </label>
      <button type="submit" :class="btnPrimary" :disabled="creating">
        {{ creating ? $t('tokens.creating') : $t('tokens.submit') }}
      </button>
    </form>

    <p v-if="error" class="mb-3 text-sm text-warn">{{ error }}</p>
    <PageState :loading="loading">
      <ul class="divide-y divide-line overflow-hidden rounded-md border border-line bg-white">
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
            <p class="mt-1 text-xs text-ink-muted">{{ formatDate(token.created_at) }}</p>
            <p v-if="token.last_used_at" class="mt-0.5 text-xs text-ink-muted">
              {{ $t('tokens.lastUsed') }}: {{ formatDate(token.last_used_at) }}
            </p>
          </div>
          <button type="button" :class="btnDanger" @click="revoke(token.id)">
            {{ $t('tokens.revoke') }}
          </button>
        </li>
        <li v-if="!tokens.length" class="px-4 py-10 text-center text-sm text-ink-muted">
          {{ $t('tokens.empty') }}
        </li>
      </ul>
    </PageState>
  </div>
</template>
