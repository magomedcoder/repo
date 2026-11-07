<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { activityApi, teamsApi, usersApi } from '@/api'
import type { ActivityEvent, Profile, Team } from '@/api/types'
import ActivityFeed from '@/components/ActivityFeed.vue'
import PageState from '@/components/ui/PageState.vue'
import { useAuth } from '@/composables/useAuth'
import { i18n, localizeError } from '@/i18n'
import Icon from '@/components/ui/Icon.vue'
import GlassTable from '@/components/ui/GlassTable.vue'

const auth = useAuth()
const profile = ref<Profile | null>(null)
const error = ref('')
const loading = ref(true)
const uploading = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const activity = ref<ActivityEvent[]>([])
const activityError = ref('')
const activityLoading = ref(true)
const teams = ref<Team[]>([])
const teamsError = ref('')
const teamsLoading = ref(true)

const avatarBust = ref(0)
const avatarSrc = computed(() => {
  const u = auth.user.value
  if (!u?.has_avatar) {
    return ''
  }
  const q = avatarBust.value ? `?t=${avatarBust.value}` : ''
  return `${usersApi.avatarURL(u.username)}${q}`
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    const username = auth.user.value?.username
    if (!username) {
      throw new Error('unauthorized')
    }

    profile.value = await usersApi.get(username)

    teamsLoading.value = true
    teamsError.value = ''
    try {
      const teamRes = await teamsApi.list()
      teams.value = (teamRes.teams ?? []).slice().sort((a, b) => a.name.localeCompare(b.name))
    } catch (err) {
      teamsError.value = localizeError(err, 'errors.loadFailed')
      teams.value = []
    } finally {
      teamsLoading.value = false
    }

    activityLoading.value = true
    activityError.value = ''
    try {
      const activityRes = await activityApi.listOwner(username)
      activity.value = activityRes.events ?? []
    } catch (err) {
      activityError.value = localizeError(err, 'errors.loadActivity')
      activity.value = []
    } finally {
      activityLoading.value = false
    }
  } catch (err) {
    error.value = localizeError(err, 'errors.loadProfile')
  } finally {
    loading.value = false
  }
}

function joinedDate(value: string) {
  const tag = i18n.global.locale.value === 'ru' ? 'ru-RU' : 'en-GB'
  return new Date(value).toLocaleDateString(tag, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  })
}

async function onAvatarSelected(ev: Event) {
  const input = ev.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  uploading.value = true
  error.value = ''
  try {
    profile.value = await usersApi.uploadAvatar(file)
    avatarBust.value = Date.now()
    if (auth.user.value) {
      auth.setUser({
        ...auth.user.value,
        has_avatar: true,
      })
    }
  } catch (err) {
    error.value = localizeError(err, 'errors.uploadAvatar')
  } finally {
    uploading.value = false
    input.value = ''
  }
}

async function removeAvatar() {
  uploading.value = true
  error.value = ''
  try {
    profile.value = await usersApi.deleteAvatar()
    avatarBust.value = Date.now()
    if (auth.user.value) {
      auth.setUser({
        ...auth.user.value,
        has_avatar: false,
      })
    }
  } catch (err) {
    error.value = localizeError(err, 'errors.uploadAvatar')
  } finally {
    uploading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div>
    <PageState :loading="loading" :error="error">
      <div v-if="profile" class="space-y-6">
        <section class="glass px-4 py-5 sm:px-5">
          <div class="flex flex-col gap-4 sm:flex-row sm:items-center">
            <div class="flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-full border border-line bg-white/55 text-2xl font-semibold text-ink backdrop-blur-md">
              <img
                v-if="avatarSrc"
                :src="avatarSrc"
                alt=""
                class="h-full w-full object-cover"
              >
              <span v-else>{{ profile.username.slice(0, 1).toUpperCase() }}</span>
            </div>
            <div class="min-w-0 flex-1 space-y-1">
              <h1 class="truncate text-[160%] font-bold leading-tight text-ink">{{ profile.username }}</h1>
              <p v-if="profile.email" class="truncate text-ink-muted">{{ profile.email }}</p>
              <p class="text-[85%] text-ink-muted">
                {{ $t('profile.memberSince', { date: joinedDate(profile.created_at) }) }}
              </p>
              <div class="flex flex-wrap gap-2 pt-2">
                <button
                  type="button"
                  class="btn"
                  :disabled="uploading"
                  @click="fileInput?.click()"
                ><Icon name="pencil" size="sm" />{{ uploading ? $t('profile.uploading') : $t('profile.changeAvatar') }}</button>
                <button
                  v-if="profile.has_avatar"
                  type="button"
                  class="btn-warn"
                  :disabled="uploading"
                  @click="removeAvatar"
                ><Icon name="trash" size="sm" />{{ $t('profile.removeAvatar') }}</button>
                <input
                  ref="fileInput"
                  type="file"
                  accept="image/png,image/jpeg,image/webp,image/gif"
                  class="hidden"
                  @change="onAvatarSelected"
                >
              </div>
            </div>
            <nav class="flex shrink-0 flex-col gap-1.5 text-[90%] sm:items-end">
              <RouterLink
                :to="{ name: 'profile-email' }"
                class="inline-flex items-center gap-1.5 text-link no-underline hover:underline"
              ><Icon name="mail" size="sm" />{{ $t('profile.updateEmail') }}</RouterLink>
              <RouterLink
                :to="{ name: 'profile-password' }"
                class="inline-flex items-center gap-1.5 text-link no-underline hover:underline"
              ><Icon name="lock" size="sm" />{{ $t('profile.changePasswordTitle') }}</RouterLink>
              <RouterLink
                :to="{ name: 'sessions' }"
                class="inline-flex items-center gap-1.5 text-link no-underline hover:underline"
              ><Icon name="device" size="sm" />{{ $t('nav.sessions') }}</RouterLink>
              <RouterLink
                :to="{ name: 'tokens' }"
                class="inline-flex items-center gap-1.5 text-link no-underline hover:underline"
              ><Icon name="key" size="sm" />{{ $t('nav.accessTokens') }}</RouterLink>
              <RouterLink
                :to="{ name: 'teams' }"
                class="inline-flex items-center gap-1.5 text-link no-underline hover:underline"
              ><Icon name="users" size="sm" />{{ $t('nav.teams') }}</RouterLink>
            </nav>
          </div>
        </section>

        <section>
          <div class="mb-2 flex items-baseline justify-between gap-3 border-b border-white/40 pb-1.5">
            <h2 class="inline-flex items-center gap-1.5 font-bold text-ink"><Icon name="users" size="sm" class="text-ink-muted" />{{ $t('profile.teams') }}</h2>
            <span class="text-[85%] text-ink-muted">{{ teams.length }}</span>
          </div>
          <p v-if="teamsLoading" class="text-ink-muted">{{ $t('common.loading') }}</p>
          <p v-else-if="teamsError" class="text-warn">{{ teamsError }}</p>
          <GlassTable v-else-if="teams.length">
            <thead>
              <tr>
                <th>{{ $t('index.name') }}</th>
                <th class="hidden sm:table-cell">{{ $t('teams.slug') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="team in teams" :key="team.slug">
                <td class="truncate">
                  <RouterLink :to="{ name: 'team', params: { slug: team.slug } }" class="inline-flex items-center gap-1.5"><Icon name="users" size="sm" class="text-ink-muted" />{{ team.name }}</RouterLink>
                </td>
                <td class="hidden truncate font-mono text-[90%] text-ink-muted sm:table-cell">
                  {{ team.slug }}
                </td>
              </tr>
            </tbody>
          </GlassTable>
          <p v-else class="text-ink-muted">{{ $t('profile.noTeams') }}</p>
        </section>

        <ActivityFeed
          :events="activity"
          :loading="activityLoading"
          :error="activityError"
        />
      </div>
    </PageState>
  </div>
</template>
