<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { activityApi, foldersApi, usersApi } from '@/api'
import type { ActivityEvent, Folder, Profile } from '@/api/types'
import ActivityFeed from '@/components/ActivityFeed.vue'
import PageState from '@/components/ui/PageState.vue'
import { useAuth } from '@/composables/useAuth'
import { folderLocation } from '@/lib/folderNav'
import { i18n, localizeError } from '@/i18n'

const auth = useAuth()
const profile = ref<Profile | null>(null)
const error = ref('')
const loading = ref(true)
const uploading = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const activity = ref<ActivityEvent[]>([])
const activityError = ref('')
const activityLoading = ref(true)
const folders = ref<Folder[]>([])
const foldersError = ref('')
const foldersLoading = ref(true)

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

    foldersLoading.value = true
    foldersError.value = ''
    try {
      const folderRes = await foldersApi.list()
      folders.value = (folderRes.folders ?? []).slice().sort((a, b) => a.path.localeCompare(b.path))
    } catch (err) {
      foldersError.value = localizeError(err, 'errors.loadFailed')
      folders.value = []
    } finally {
      foldersLoading.value = false
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

function folderRoleLabel(folder: Folder) {
  const role = folder.role
  if (role === 'owner' || role === 'admin' || role === 'member') {
    return `folderAccess.roles.${role}`
  }
  return ''
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
        <section class="border border-line bg-path px-4 py-5 sm:px-5">
          <div class="flex flex-col gap-4 sm:flex-row sm:items-center">
            <div class="flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-full border border-line bg-paper text-2xl font-semibold text-ink">
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
                  class="inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-ink hover:bg-row-hover disabled:cursor-not-allowed disabled:opacity-50"
                  :disabled="uploading"
                  @click="fileInput?.click()"
                >{{ uploading ? $t('profile.uploading') : $t('profile.changeAvatar') }}</button>
                <button
                  v-if="profile.has_avatar"
                  type="button"
                  class="inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-ink hover:bg-row-hover disabled:cursor-not-allowed disabled:opacity-50"
                  :disabled="uploading"
                  @click="removeAvatar"
                >{{ $t('profile.removeAvatar') }}</button>
                <input
                  ref="fileInput"
                  type="file"
                  accept="image/png,image/jpeg,image/webp,image/gif"
                  class="hidden"
                  @change="onAvatarSelected"
                >
              </div>
            </div>
            <nav class="flex shrink-0 flex-col gap-1 text-[90%] sm:items-end">
              <RouterLink
                :to="{ name: 'profile-email' }"
                class="text-link hover:underline"
              >{{ $t('profile.updateEmail') }}</RouterLink>
              <RouterLink
                :to="{ name: 'profile-password' }"
                class="text-link hover:underline"
              >{{ $t('profile.changePasswordTitle') }}</RouterLink>
              <RouterLink
                :to="{ name: 'sessions' }"
                class="text-link hover:underline"
              >{{ $t('nav.sessions') }}</RouterLink>
              <RouterLink
                :to="{ name: 'tokens' }"
                class="text-link hover:underline"
              >{{ $t('nav.accessTokens') }}</RouterLink>
            </nav>
          </div>
        </section>

        <section>
          <div class="mb-2 flex items-baseline justify-between gap-3 border-b border-line pb-1.5">
            <h2 class="font-bold text-ink">{{ $t('profile.folders') }}</h2>
            <span class="text-[85%] text-ink-muted">{{ folders.length }}</span>
          </div>
          <p v-if="foldersLoading" class="text-ink-muted">{{ $t('common.loading') }}</p>
          <p v-else-if="foldersError" class="text-warn">{{ foldersError }}</p>
          <table
            v-else-if="folders.length"
            class="w-full border-collapse [&_th]:px-2 [&_th]:py-1 [&_th]:text-left [&_th]:align-baseline [&_th]:font-medium [&_th]:text-ink-muted [&_td]:border-0 [&_td]:px-2 [&_td]:py-1.5 [&_td]:align-baseline [&_tbody_tr:nth-child(odd)]:bg-paper [&_tbody_tr:nth-child(even)]:bg-row-alt [&_tbody_tr:hover]:bg-row-hover"
          >
            <thead>
              <tr>
                <th>{{ $t('index.name') }}</th>
                <th class="hidden sm:table-cell">{{ $t('index.path') }}</th>
                <th class="w-[22%]">{{ $t('profile.folderRole') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="folder in folders" :key="folder.path">
                <td class="truncate">
                  <RouterLink :to="folderLocation(folder)">{{ folder.name }}</RouterLink>
                </td>
                <td class="hidden truncate font-mono text-[90%] text-ink-muted sm:table-cell">
                  {{ folder.path }}
                </td>
                <td class="whitespace-nowrap text-ink-muted">
                  <template v-if="folderRoleLabel(folder)">
                    {{ $t(folderRoleLabel(folder)) }}
                  </template>
                </td>
              </tr>
            </tbody>
          </table>
          <p v-else class="text-ink-muted">{{ $t('profile.noFolders') }}</p>
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
