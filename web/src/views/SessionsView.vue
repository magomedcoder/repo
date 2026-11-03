<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '@/api'
import type { AuthSession } from '@/api/types'
import PageState from '@/components/ui/PageState.vue'
import { useAuth } from '@/composables/useAuth'
import { formatDate, i18n, localizeError } from '@/i18n'

const router = useRouter()
const auth = useAuth()
const sessions = ref<AuthSession[]>([])
const error = ref('')
const loading = ref(true)
const busy = ref(false)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await authApi.listSessions()
    sessions.value = res.sessions ?? []
  } catch (err) {
    error.value = localizeError(err, 'errors.loadSessions')
  } finally {
    loading.value = false
  }
}

async function revoke(session: AuthSession) {
  const key = session.current ? 'sessions.confirmRevokeCurrent' : 'sessions.confirmRevoke'
  if (!confirm(i18n.global.t(key))) {
    return
  }

  busy.value = true
  error.value = ''
  try {
    await authApi.revokeSession(session.id)
    if (session.current) {
      auth.setUser(null)
      await router.push({ name: 'login' })
      return
    }
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.deleteFailed')
  } finally {
    busy.value = false
  }
}

async function revokeOthers() {
  if (!confirm(i18n.global.t('sessions.confirmRevokeOthers'))) {
    return
  }

  busy.value = true
  error.value = ''
  try {
    await authApi.revokeOtherSessions()
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.deleteFailed')
  } finally {
    busy.value = false
  }
}

async function revokeAll() {
  if (!confirm(i18n.global.t('sessions.confirmRevokeAll'))) {
    return
  }

  busy.value = true
  error.value = ''
  try {
    await authApi.revokeAllSessions()
    auth.setUser(null)
    await router.push({ name: 'login' })
  } catch (err) {
    error.value = localizeError(err, 'errors.deleteFailed')
    busy.value = false
  }
}

function deviceLabel(session: AuthSession): string {
  const ua = (session.user_agent || '').trim()
  if (!ua) {
    return i18n.global.t('sessions.unknownDevice')
  }
  if (ua.length > 96) {
    return `${ua.slice(0, 96)}...`
  }
  return ua
}

onMounted(load)
</script>

<template>
  <div class="mx-auto max-w-3xl">
    <div class="mb-4 flex flex-wrap items-start justify-between gap-3 border-b border-line pb-3">
      <div>
        <h1 class="text-[140%] font-bold text-ink">{{ $t('sessions.title') }}</h1>
        <p class="mt-1 text-sm text-ink-muted">{{ $t('sessions.hint') }}</p>
      </div>
      <div class="flex flex-wrap gap-2">
        <button
          type="button"
          class="inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-ink hover:bg-row-hover disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="busy || sessions.length < 2"
          @click="revokeOthers"
        >
          {{ $t('sessions.revokeOthers') }}
        </button>
        <button
          type="button"
          class="inline-flex items-center justify-center border border-warn bg-paper px-2.5 py-1 text-[10pt] text-warn hover:bg-row-hover disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="busy || sessions.length === 0"
          @click="revokeAll"
        >
          {{ $t('sessions.revokeAll') }}
        </button>
      </div>
    </div>

    <PageState :loading="loading" :error="error">
      <p v-if="!sessions.length" class="text-sm text-ink-muted">{{ $t('sessions.empty') }}</p>
      <ul v-else class="divide-y divide-line border border-line bg-paper">
        <li
          v-for="session in sessions"
          :key="session.id"
          class="flex flex-col gap-2 px-3 py-3 sm:flex-row sm:items-center sm:justify-between sm:px-4"
        >
          <div class="min-w-0 space-y-1">
            <div class="flex flex-wrap items-center gap-2">
              <p class="truncate text-sm font-medium text-ink" :title="session.user_agent">
                {{ deviceLabel(session) }}
              </p>
              <span
                v-if="session.current"
                class="border border-line bg-path px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-ink-muted"
              >
                {{ $t('sessions.current') }}
              </span>
            </div>
            <p class="text-xs text-ink-muted">
              <span v-if="session.ip">{{ session.ip }} </span>
              {{ $t('sessions.lastSeen') }}: {{ formatDate(session.last_seen_at) }}
              {{ $t('sessions.created') }}: {{ formatDate(session.created_at) }}
            </p>
          </div>
          <button
            type="button"
            class="inline-flex shrink-0 items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-ink hover:bg-row-hover disabled:cursor-not-allowed disabled:opacity-50"
            :disabled="busy"
            @click="revoke(session)"
          >
            {{ session.current ? $t('sessions.signOut') : $t('sessions.revoke') }}
          </button>
        </li>
      </ul>
    </PageState>
  </div>
</template>
