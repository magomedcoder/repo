<script setup lang="ts">
import { btnGhost, btnPrimary, field } from '@/lib/classes'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { activityApi, authApi, usersApi } from '@/api'
import type { ActivityEvent, Profile } from '@/api/types'
import ActivityFeed from '@/components/ActivityFeed.vue'
import PageState from '@/components/ui/PageState.vue'
import { useAuth } from '@/composables/useAuth'
import { useBreadcrumbs } from '@/composables/useBreadcrumbs'
import { localizeError } from '@/i18n'

const auth = useAuth()
const { setBreadcrumbs, clearBreadcrumbs } = useBreadcrumbs()

const profile = ref<Profile | null>(null)
const email = ref('')
const error = ref('')
const loading = ref(true)
const saving = ref(false)
const currentPassword = ref('')
const nextPassword = ref('')
const passwordMessage = ref('')
const uploading = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const activity = ref<ActivityEvent[]>([])
const activityError = ref('')
const activityLoading = ref(true)

const avatarSrc = computed(() => {
  const u = auth.user.value
  if (!u?.has_avatar) {
    return ''
  }
  
  return `${usersApi.avatarURL(u.username)}?t=${Date.now()}`
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    setBreadcrumbs([{ 
      label: 'profile', 
      labelKey: 'nav.profile' 
    }])
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
  error.value = ''
  try {
    profile.value = await usersApi.updateMe({ 
      email: email.value 
    })
    if (auth.user.value) {
      auth.setUser({
        ...auth.user.value,
        email: profile.value.email || email.value,
        has_avatar: profile.value.has_avatar,
      })
    }
  } catch (err) {
    error.value = localizeError(err, 'errors.saveProfile')
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
  passwordMessage.value = ''
  error.value = ''
  try {
    await authApi.password({
      current: currentPassword.value,
      next: nextPassword.value,
    })
    currentPassword.value = ''
    nextPassword.value = ''
    passwordMessage.value = 'ok'
  } catch (err) {
    error.value = localizeError(err, 'errors.saveFailed')
  }
}

onMounted(load)
onUnmounted(clearBreadcrumbs)
</script>

<template>
  <div class="mx-auto max-w-xl">
    <h1 class="text-2xl font-semibold">{{ $t('profile.title') }}</h1>
    <p class="mt-1 text-sm text-ink-muted">{{ $t('profile.lead') }}</p>

    <PageState :loading="loading" :error="error">
      <div v-if="profile" class="mt-6 space-y-6">
        <div class="flex items-center gap-4">
          <div class="flex h-16 w-16 items-center justify-center overflow-hidden rounded-full bg-paper-2 text-xl font-semibold text-ink-muted">
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
                :class="btnGhost"
                :disabled="uploading"
                @click="fileInput?.click()"
              >{{ uploading ? $t('profile.uploading') : $t('profile.changeAvatar') }}</button>
              <button
                v-if="profile.has_avatar"
                type="button"
                :class="btnGhost"
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
              :class="[field, 'w-full']"
            >
          </label>
          <button type="submit" :class="btnPrimary" :disabled="saving">
            {{ saving ? $t('profile.saving') : $t('profile.save') }}
          </button>
        </form>

        <form class="space-y-3" @submit.prevent="changePassword">
          <label class="block text-sm">
            <span class="mb-1 block font-semibold">{{ $t('auth.password') }}</span>
            <input v-model="currentPassword" type="password" required :class="field" autocomplete="current-password" />
          </label>
          <label class="block text-sm">
            <span class="mb-1 block font-semibold">{{ $t('profile.newPassword') }}</span>
            <input v-model="nextPassword" type="password" required minlength="8" :class="field" autocomplete="new-password" />
          </label>
          <button type="submit" :class="btnPrimary">{{ $t('profile.save') }}</button>
          <p v-if="passwordMessage">{{ $t('common.ok') }}</p>
        </form>

        <p class="flex flex-wrap gap-x-4 gap-y-1 text-sm">
          <RouterLink
            :to="{ name: 'tokens' }"
            class="text-accent hover:underline"
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
