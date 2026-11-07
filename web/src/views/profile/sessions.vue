<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '@/api'
import type { AuthSession } from '@/api/types'
import PageState from '@/components/ui/PageState.vue'
import { useAuth } from '@/composables/useAuth'
import { formatDate, i18n, localizeError } from '@/i18n'
import Icon from '@/components/ui/Icon.vue'
import PageHeader from '@/components/ui/PageHeader.vue'
import Surface from '@/components/ui/Surface.vue'

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
    <PageHeader :title="$t('sessions.title')" icon="device" :hint="$t('sessions.hint')">
      <template #actions>
        <button
          type="button"
          class="btn"
          :disabled="busy || sessions.length < 2"
          @click="revokeOthers"
        >
          {{ $t('sessions.revokeOthers') }}
        </button>
        <button
          type="button"
          class="btn-warn"
          :disabled="busy || sessions.length === 0"
          @click="revokeAll"
        >
          {{ $t('sessions.revokeAll') }}
        </button>
      </template>
    </PageHeader>

    <PageState :loading="loading" :error="error">
      <p v-if="!sessions.length" class="text-sm text-ink-muted">{{ $t('sessions.empty') }}</p>
      <Surface
        v-else
        as="ul"
        class="divide-y divide-line"
      >
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
                class="rounded-md border border-line bg-white/45 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-ink-muted"
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
            class="btn shrink-0"
            :disabled="busy"
            @click="revoke(session)"
          >
            {{ session.current ? $t('sessions.signOut') : $t('sessions.revoke') }}
          </button>
        </li>
      </Surface>
    </PageState>
  </div>
</template>
