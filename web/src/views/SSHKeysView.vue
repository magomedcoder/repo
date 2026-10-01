<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { sshKeysApi } from '@/api'
import type { SSHKey } from '@/api/types'
import PageState from '@/components/ui/PageState.vue'
import { useBreadcrumbs } from '@/composables/useBreadcrumbs'
import { formatDate, i18n, localizeError } from '@/i18n'

const { setBreadcrumbs, clearBreadcrumbs } = useBreadcrumbs()

const keys = ref<SSHKey[]>([])
const title = ref('')
const publicKey = ref('')
const error = ref('')
const loading = ref(true)
const creating = ref(false)
const showForm = ref(false)

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
        label: 'ssh',
        labelKey: 'nav.sshKeys'
      }
    ])
    const res = await sshKeysApi.list()
    keys.value = res.keys ?? []
  } catch (err) {
    error.value = localizeError(err, 'errors.loadSSHKeys')
  } finally {
    loading.value = false
  }
}

async function create() {
  creating.value = true
  error.value = ''
  try {
    await sshKeysApi.create({ 
      title: title.value, 
      public_key: publicKey.value
    })
    title.value = ''
    publicKey.value = ''
    showForm.value = false
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.createSSHKey')
  } finally {
    creating.value = false
  }
}

async function remove(id: number) {
  if (!confirm(i18n.global.t('sshKeys.confirmDelete'))) {
    return
  }

  error.value = ''
  try {
    await sshKeysApi.remove(id)
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.deleteFailed')
  }
}

onMounted(load)
onUnmounted(clearBreadcrumbs)
</script>

<template>
  <div class="mx-auto max-w-3xl px-4 py-6">
    <div class="mb-4 flex flex-wrap items-start justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold">{{ $t('sshKeys.title') }}</h1>
        <p class="mt-1 text-sm text-ink-muted">{{ $t('sshKeys.lead') }}</p>
      </div>
      <button
        type="button" 
        class="btn-primary" 
        @click="showForm = !showForm"
      >{{ $t('sshKeys.add') }}</button>
    </div>

    <form 
      v-if="showForm" 
      class="mb-4 space-y-3 rounded-md border border-line bg-white p-4" 
      @submit.prevent="create"
    >
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('sshKeys.titleField') }}</span>
        <input 
          v-model="title" 
          class="input" 
          :placeholder="$t('sshKeys.titlePlaceholder')" 
        />
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('sshKeys.keyField') }}</span>
        <textarea
          v-model="publicKey"
          class="input min-h-28 font-mono text-xs"
          required
          :placeholder="$t('sshKeys.keyPlaceholder')"
        />
      </label>
      <button 
        type="submit" 
        class="btn-primary" 
        :disabled="creating"
      >
        {{ creating ? $t('sshKeys.adding') : $t('sshKeys.submit') }}
      </button>
    </form>

    <p v-if="error" class="mb-3 text-sm text-warn">{{ error }}</p>
    <PageState :loading="loading">
      <ul class="divide-y divide-line overflow-hidden rounded-md border border-line bg-white">
        <li 
          v-for="key in keys" 
          :key="key.id" 
          class="flex flex-wrap items-start justify-between gap-3 px-4 py-3"
        >
          <div class="min-w-0">
            <p class="font-semibold">{{ key.title }}</p>
            <p class="mt-1 break-all font-mono text-xs text-ink-muted">
              {{ $t('sshKeys.fingerprint') }}: {{ key.fingerprint }}
            </p>
            <p class="mt-1 text-xs text-ink-muted">{{ formatDate(key.created_at) }}</p>
          </div>
          <button 
            type="button" 
            class="btn-danger" 
            @click="remove(key.id)"
          >{{ $t('sshKeys.delete') }}</button>
        </li>
        <li v-if="!keys.length" class="px-4 py-10 text-center text-sm text-ink-muted">{{ $t('sshKeys.empty') }}</li>
      </ul>
    </PageState>
  </div>
</template>
