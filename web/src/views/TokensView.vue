<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { tokensApi } from '@/api'
import type { AccessToken } from '@/api/types'
import PageState from '@/components/ui/PageState.vue'
import { formatDate, i18n, localizeError } from '@/i18n'

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
</script>

<template>
  <div class="mx-auto max-w-3xl">
    <div class="mb-4 flex flex-wrap items-start justify-between gap-3 border-b border-line pb-3">
      <div>
        <h1 class="text-[140%] font-bold text-ink">{{ $t('tokens.title') }}</h1>
      </div>
      <button type="button" class="inline-flex items-center justify-center border border-ink bg-ink px-2.5 py-1 text-[10pt] text-paper hover:bg-ink-muted disabled:cursor-not-allowed disabled:opacity-50" @click="showForm = !showForm">
        {{ $t('tokens.add') }}
      </button>
    </div>

    <div
      v-if="createdToken"
      class="overflow-hidden border border-line bg-paper mb-4"
    >
      <div class="flex items-center gap-2 border-b border-line bg-path px-3 py-1.5 text-[10pt] text-ink">{{ $t('tokens.createdTitle') }}</div>
      <div class="p-3 sm:p-4">
        <p class="text-xs text-ink-muted">{{ $t('tokens.createdHint') }}</p>
        <pre class="mt-2 overflow-x-auto border border-line bg-paper-2 p-3 font-mono text-xs">{{ createdToken }}</pre>
        <div class="mt-3 flex flex-wrap gap-2">
          <button type="button" class="inline-flex items-center justify-center border border-ink bg-ink px-2.5 py-1 text-[10pt] text-paper hover:bg-ink-muted disabled:cursor-not-allowed disabled:opacity-50" @click="copyToken">{{ $t('common.copy') }}</button>
          <button type="button" class="inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-ink hover:bg-row-hover disabled:cursor-not-allowed disabled:opacity-50" @click="createdToken = ''">{{ $t('common.close') }}</button>
        </div>
      </div>
    </div>

    <form
      v-if="showForm"
      class="overflow-hidden border border-line bg-paper mb-4"
      @submit.prevent="create"
    >
      <div class="flex items-center gap-2 border-b border-line bg-path px-3 py-1.5 text-[10pt] text-ink">{{ $t('tokens.add') }}</div>
      <div class="p-3 sm:p-4 space-y-3">
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('tokens.nameField') }}</span>
          <input
            v-model="name"
            class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none"
            required
            maxlength="64"
            :placeholder="$t('tokens.namePlaceholder')"
          />
        </label>
        <button type="submit" class="inline-flex items-center justify-center border border-ink bg-ink px-2.5 py-1 text-[10pt] text-paper hover:bg-ink-muted disabled:cursor-not-allowed disabled:opacity-50" :disabled="creating">
          {{ creating ? $t('tokens.creating') : $t('tokens.submit') }}
        </button>
      </div>
    </form>

    <p v-if="error" class="mb-3 text-sm text-warn">{{ error }}</p>
    <PageState :loading="loading">
      <ul class="overflow-hidden border border-line bg-paper divide-y divide-line">
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
          <button type="button" class="inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-warn hover:bg-warn-soft disabled:cursor-not-allowed disabled:opacity-50" @click="revoke(token.id)">
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
