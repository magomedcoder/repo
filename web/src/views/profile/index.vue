<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { activityApi, reposApi, teamsApi, usersApi } from '@/api'
import type { ActivityEvent, Profile, Repository, Team } from '@/api/types'
import ActivityFeed from '@/components/ActivityFeed.vue'
import ContentsList from '@/components/ContentsList.vue'
import Icon from '@/components/ui/Icon.vue'
import PageState from '@/components/ui/PageState.vue'
import Surface from '@/components/ui/Surface.vue'
import { useAuth } from '@/composables/useAuth'
import { formatDate, i18n, localizeError } from '@/i18n'

const PROFILE_REPO_LIMIT = 30

const auth = useAuth()
const profile = ref<Profile | null>(null)
const error = ref('')
const loading = ref(true)
const uploading = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

const repos = ref<Repository[]>([])
const reposError = ref('')
const reposLoading = ref(true)

const teams = ref<Team[]>([])
const teamsError = ref('')
const teamsLoading = ref(true)

const activity = ref<ActivityEvent[]>([])
const activityError = ref('')
const activityLoading = ref(true)

const avatarBust = ref(0)
const avatarSrc = computed(() => {
  const u = auth.user.value
  if (!u?.has_avatar) {
    return ''
  }
  const q = avatarBust.value ? `?t=${avatarBust.value}` : ''
  return `${usersApi.avatarURL(u.username)}${q}`
})

async function loadRepos() {
  reposLoading.value = true
  reposError.value = ''
  try {
    const [ownRes, sharedRes] = await Promise.all([
      reposApi.listOwn('idle', { offset: 0, limit: PROFILE_REPO_LIMIT }),
      reposApi.listShared('idle', { offset: 0, limit: PROFILE_REPO_LIMIT }),
    ])
    const own = ownRes.repos ?? []
    const seen = new Set(own.map((r) => r.id))
    const shared = (sharedRes.repos ?? []).filter((r) => !seen.has(r.id))
    repos.value = [...own, ...shared]
  } catch (err) {
    reposError.value = localizeError(err, 'errors.loadFailed')
    repos.value = []
  } finally {
    reposLoading.value = false
  }
}

async function loadTeams() {
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
}

async function loadActivity(username: string) {
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
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const username = auth.user.value?.username
    if (!username) {
      throw new Error('unauthorized')
    }

    profile.value = await usersApi.get(username)
    await Promise.all([loadRepos(), loadTeams(), loadActivity(username)])
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
  if (!file) {
    return
  }

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
  <div class="page-enter">
    <PageState :loading="loading" :error="error">
      <div v-if="profile" class="space-y-5">
        <section class="surface overflow-hidden">
          <div class="relative overflow-hidden px-4 py-5 sm:px-5">
            <div
              class="pointer-events-none absolute inset-0 bg-[radial-gradient(520px_220px_at_12%_-10%,rgba(255,255,255,0.55),transparent_60%),radial-gradient(420px_200px_at_90%_0%,rgba(168,184,204,0.35),transparent_55%)]"
              aria-hidden="true"
            />
            <div class="relative flex flex-col gap-4 sm:flex-row sm:items-center">
              <div class="flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-full border border-line-strong/70 bg-white/65 text-2xl font-semibold text-ink shadow-[inset_0_1px_0_rgba(255,255,255,0.7)]">
                <img
                  v-if="avatarSrc"
                  :src="avatarSrc"
                  alt=""
                  class="h-full w-full object-cover"
                >
                <span v-else>{{ profile.username.slice(0, 1).toUpperCase() }}</span>
              </div>
              <div class="min-w-0 flex-1 space-y-1">
                <h1 class="truncate text-[1.55rem] font-semibold leading-tight text-ink">
                  {{ profile.username }}
                </h1>
                <p v-if="profile.email" class="truncate text-[13px] text-ink-muted">{{ profile.email }}</p>
                <p class="text-[12px] text-ink-muted">
                  {{ $t('profile.memberSince', { date: joinedDate(profile.created_at) }) }}
                </p>
                <div class="flex flex-wrap gap-2 pt-2">
                  <button
                    type="button"
                    class="btn"
                    :disabled="uploading"
                    @click="fileInput?.click()"
                  >
                    <Icon name="pencil" size="sm" />
                    {{ uploading ? $t('profile.uploading') : $t('profile.changeAvatar') }}
                  </button>
                  <button
                    v-if="profile.has_avatar"
                    type="button"
                    class="btn-warn"
                    :disabled="uploading"
                    @click="removeAvatar"
                  >
                    <Icon name="trash" size="sm" />
                    {{ $t('profile.removeAvatar') }}
                  </button>
                  <input
                    ref="fileInput"
                    type="file"
                    accept="image/png,image/jpeg,image/webp,image/gif"
                    class="hidden"
                    @change="onAvatarSelected"
                  >
                </div>
              </div>
            </div>
          </div>
          <div class="flex flex-wrap gap-1.5 border-t border-line-strong/35 bg-white/25 px-3 py-2.5 sm:px-4">
            <span class="mr-1 self-center text-[11px] font-semibold uppercase tracking-wide text-ink-muted">
              {{ $t('profile.settings') }}
            </span>
            <RouterLink :to="{ name: 'profile-email' }" class="btn px-2 py-1 text-[12px] no-underline">
              <Icon name="mail" size="sm" />
              {{ $t('profile.updateEmail') }}
            </RouterLink>
            <RouterLink :to="{ name: 'profile-password' }" class="btn px-2 py-1 text-[12px] no-underline">
              <Icon name="lock" size="sm" />
              {{ $t('profile.changePasswordTitle') }}
            </RouterLink>
            <RouterLink :to="{ name: 'sessions' }" class="btn px-2 py-1 text-[12px] no-underline">
              <Icon name="device" size="sm" />
              {{ $t('nav.sessions') }}
            </RouterLink>
            <RouterLink :to="{ name: 'tokens' }" class="btn px-2 py-1 text-[12px] no-underline">
              <Icon name="key" size="sm" />
              {{ $t('nav.accessTokens') }}
            </RouterLink>
            <RouterLink :to="{ name: 'teams' }" class="btn px-2 py-1 text-[12px] no-underline">
              <Icon name="users" size="sm" />
              {{ $t('nav.teams') }}
            </RouterLink>
          </div>
        </section>

        <div class="grid gap-5 lg:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)]">
          <div class="space-y-5">
            <p v-if="reposError" class="text-sm text-warn">{{ reposError }}</p>
            <p v-else-if="reposLoading" class="text-ink-muted">{{ $t('common.loading') }}</p>
            <ContentsList
              v-else
              :repos="repos"
              :title="$t('profile.repositories')"
              :hint="String(repos.length)"
              :empty-text="$t('profile.noRepositories')"
            />
          </div>

          <div class="space-y-5">
            <Surface class="overflow-hidden">
              <div class="flex items-center justify-between gap-2 border-b border-line-strong/40 px-3 py-2 sm:px-4">
                <h2 class="inline-flex items-center gap-1.5 text-[13px] font-semibold text-ink">
                  <Icon name="users" size="sm" class="text-ink-muted" />
                  {{ $t('profile.teams') }}
                </h2>
                <div class="flex items-center gap-2">
                  <span class="text-[12px] text-ink-muted">{{ teams.length }}</span>
                  <RouterLink
                    :to="{ name: 'teams' }"
                    class="text-[12px] text-link no-underline hover:underline"
                  >
                    {{ $t('profile.manageTeams') }}
                  </RouterLink>
                </div>
              </div>
              <p v-if="teamsLoading" class="px-3 py-4 text-sm text-ink-muted sm:px-4">
                {{ $t('common.loading') }}
              </p>
              <p v-else-if="teamsError" class="px-3 py-4 text-sm text-warn sm:px-4">
                {{ teamsError }}
              </p>
              <ul v-else-if="teams.length" class="divide-y divide-line-strong/30">
                <li v-for="team in teams" :key="team.slug">
                  <RouterLink
                    :to="{ name: 'team', params: { slug: team.slug } }"
                    class="flex items-center justify-between gap-3 px-3 py-2.5 text-ink no-underline transition-colors hover:bg-row-hover sm:px-4"
                  >
                    <span class="inline-flex min-w-0 items-center gap-2">
                      <span class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-full border border-line-strong/70 bg-white/40 text-ink-muted">
                        <Icon name="users" size="sm" />
                      </span>
                      <span class="min-w-0">
                        <span class="block truncate text-[13px] font-semibold">{{ team.name }}</span>
                        <span class="block truncate font-mono text-[11px] text-ink-muted">{{ team.slug }}</span>
                      </span>
                    </span>
                    <span class="shrink-0 text-[11px] text-ink-muted">
                      {{ formatDate(team.created_at) }}
                    </span>
                  </RouterLink>
                </li>
              </ul>
              <p v-else class="px-3 py-5 text-center text-sm text-ink-muted sm:px-4">
                {{ $t('profile.noTeams') }}
              </p>
            </Surface>

            <ActivityFeed
              :events="activity"
              :loading="activityLoading"
              :error="activityError"
            />
          </div>
        </div>
      </div>
    </PageState>
  </div>
</template>
