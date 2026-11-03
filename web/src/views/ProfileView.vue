<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { activityApi, authApi, usersApi } from '@/api'
import type { ActivityEvent, Profile } from '@/api/types'
import ActivityFeed from '@/components/ActivityFeed.vue'
import PageState from '@/components/ui/PageState.vue'
import { useAuth } from '@/composables/useAuth'
import { localizeError } from '@/i18n'

const auth = useAuth()
const profile = ref<Profile | null>(null)
const email = ref('')
const error = ref('')
const emailError = ref('')
const emailSaved = ref(false)
const loading = ref(true)
const saving = ref(false)
const currentPassword = ref('')
const nextPassword = ref('')
const passwordError = ref('')
const passwordSaved = ref(false)
const passwordSaving = ref(false)
const uploading = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
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

async function load() {
  loading.value = true
  error.value = ''
  try {
    const username = auth.user.value?.username
    if (!username) {
      throw new Error('unauthorized')
    }

    profile.value = await usersApi.get(username)
    email.value = profile.value.email || ''
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

async function saveEmail() {
  saving.value = true
  emailError.value = ''
  emailSaved.value = false
  try {
    profile.value = await usersApi.updateMe({
      email: email.value,
    })
    if (auth.user.value) {
      auth.setUser({
        ...auth.user.value,
        email: profile.value.email || email.value,
        has_avatar: profile.value.has_avatar,
      })
    }
    emailSaved.value = true
  } catch (err) {
    emailError.value = localizeError(err, 'errors.saveProfile')
  } finally {
    saving.value = false
  }
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
        has_avatar: true 
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
        has_avatar: false 
      })
    }
  } catch (err) {
    error.value = localizeError(err, 'errors.uploadAvatar')
  } finally {
    uploading.value = false
  }
}

async function changePassword() {
  passwordError.value = ''
  passwordSaved.value = false
  passwordSaving.value = true
  try {
    await authApi.password({
      current: currentPassword.value,
      next: nextPassword.value,
    })
    currentPassword.value = ''
    nextPassword.value = ''
    passwordSaved.value = true
  } catch (err) {
    passwordError.value = localizeError(err, 'errors.saveFailed')
  } finally {
    passwordSaving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="mx-auto max-w-xl">
    <h1 class="mb-3 text-[140%] font-bold text-ink">{{ $t('profile.title') }}</h1>

    <PageState :loading="loading" :error="error">
      <div v-if="profile" class="mt-6 space-y-6">
        <div class="flex items-center gap-4">
          <div class="flex h-16 w-16 items-center justify-center overflow-hidden border border-line bg-paper-2 text-xl font-semibold text-ink">
            <img
              v-if="avatarSrc"
              :src="avatarSrc"
              alt=""
              class="h-full w-full object-cover"
            >
            <span v-else>{{ profile.username.slice(0, 1).toUpperCase() }}</span>
          </div>
          <div class="space-y-2">
            <p class="font-semibold">{{ profile.username }}</p>
            <div class="flex flex-wrap gap-2">
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
            </div>
            <input
              ref="fileInput"
              type="file"
              accept="image/png,image/jpeg,image/webp,image/gif"
              class="hidden"
              @change="onAvatarSelected"
            >
          </div>
        </div>

        <form class="space-y-3" @submit.prevent="saveEmail">
          <label class="block text-sm">
            <span class="mb-1 block font-semibold">{{ $t('profile.email') }}</span>
            <input
              v-model="email"
              type="email"
              required
              class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none"
            >
          </label>
          <button
            type="submit"
            class="inline-flex items-center justify-center border border-ink bg-ink px-2.5 py-1 text-[10pt] text-paper hover:bg-ink-muted disabled:cursor-not-allowed disabled:opacity-50"
            :disabled="saving"
          >
            {{ saving ? $t('profile.saving') : $t('profile.save') }}
          </button>
          <p v-if="emailError" class="text-sm text-warn">{{ emailError }}</p>
          <p v-else-if="emailSaved" class="text-sm text-ink-muted">{{ $t('common.ok') }}</p>
        </form>

        <form class="space-y-3" @submit.prevent="changePassword">
          <label class="block text-sm">
            <span class="mb-1 block font-semibold">{{ $t('profile.currentPassword') }}</span>
            <input
              v-model="currentPassword"
              type="password"
              required
              class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none"
              autocomplete="current-password"
            >
          </label>
          <label class="block text-sm">
            <span class="mb-1 block font-semibold">{{ $t('profile.newPassword') }}</span>
            <input
              v-model="nextPassword"
              type="password"
              required
              minlength="8"
              class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none"
              autocomplete="new-password"
            >
          </label>
          <button
            type="submit"
            class="inline-flex items-center justify-center border border-ink bg-ink px-2.5 py-1 text-[10pt] text-paper hover:bg-ink-muted disabled:cursor-not-allowed disabled:opacity-50"
            :disabled="passwordSaving"
          >
            {{ passwordSaving ? $t('profile.saving') : $t('profile.save') }}
          </button>
          <p v-if="passwordError" class="text-sm text-warn">{{ passwordError }}</p>
          <p v-else-if="passwordSaved" class="text-sm text-ink-muted">{{ $t('common.ok') }}</p>
        </form>

        <p class="flex flex-wrap gap-x-4 gap-y-1 text-sm">
          <RouterLink
            :to="{ name: 'sessions' }"
            class="text-ink hover:underline"
          >{{ $t('nav.sessions') }}</RouterLink>
          <RouterLink
            :to="{ name: 'tokens' }"
            class="text-ink hover:underline"
          >{{ $t('nav.accessTokens') }}</RouterLink>
        </p>

        <ActivityFeed
          :events="activity"
          :loading="activityLoading"
          :error="activityError"
        />
      </div>
    </PageState>
  </div>
</template>
