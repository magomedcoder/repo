<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { reposApi } from '@/api'
import type { MirrorDirection } from '@/api/types'
import ModalForm from '@/components/ui/ModalForm.vue'
import { isSSHURL } from '@/lib/mirror'
import { suggestRepoNameFromURL } from '@/lib/repoCreate'
import { i18n, localizeError } from '@/i18n'

const props = defineProps<{
  initialUrl?: string
}>()

const emit = defineEmits<{
  close: []
  created: []
}>()

const name = ref('')
const description = ref('')
const isPrivate = ref(true)
const showOnHome = ref(false)
const defaultBranch = ref('main')
const mirrorDirection = ref<MirrorDirection>('pull')
const mirrorURL = ref(props.initialUrl ?? '')
const mirrorUsername = ref('')
const mirrorPassword = ref('')
const nameTouched = ref(false)
const elapsed = ref(0)
const error = ref('')
const busy = ref(false)
let timer: ReturnType<typeof setInterval> | undefined

watch(mirrorURL, (raw) => {
  if (!nameTouched.value) {
    name.value = suggestRepoNameFromURL(raw)
  }
})

if (mirrorURL.value) {
  name.value = suggestRepoNameFromURL(mirrorURL.value)
}

const progressText = computed(() => {
  if (!busy.value) {
    return ''
  }
  const key = mirrorDirection.value === 'push' ? 'modal.progressPush' : 'modal.progressPull'
  return `${i18n.global.t(key)} ${i18n.global.t('modal.progressElapsed', { n: elapsed.value })}`
})

function stopTimer() {
  if (timer) {
    clearInterval(timer)
    timer = undefined
  }
}

onBeforeUnmount(stopTimer)

async function submit() {
  error.value = ''
  busy.value = true
  elapsed.value = 0
  stopTimer()
  timer = setInterval(() => {
    elapsed.value += 1
  }, 1000)
  try {
    const body: Parameters<typeof reposApi.create>[0] = {
      name: name.value.trim(),
      description: description.value.trim(),
      private: isPrivate.value,
      show_on_home: !isPrivate.value && showOnHome.value,
      default_branch: defaultBranch.value.trim() || 'main',
      mirror_direction: mirrorDirection.value,
      mirror_url: mirrorURL.value.trim(),
    }
    if (mirrorUsername.value.trim()) {
      body.mirror_username = mirrorUsername.value.trim()
    }

    if (mirrorPassword.value) {
      body.mirror_password = mirrorPassword.value
    }
    
    await reposApi.create(body)
    emit('created')
    emit('close')
  } catch (err) {
    error.value = localizeError(err, 'errors.createRepository')
  } finally {
    stopTimer()
    busy.value = false
  }
}
</script>

<template>
  <ModalForm
    :title="$t('modal.mirrorRepository')"
    :busy="busy"
    :submit-label="busy ? $t('modal.mirroring') : $t('modal.mirrorRepository')"
    @close="emit('close')"
    @submit="submit"
  >
    <p class="text-[85%] text-ink-muted">{{ $t('modal.mirrorModeHint') }}</p>
    <label class="block text-sm">
      <span class="mb-1 block font-medium">{{ $t('modal.mirrorDirection') }}</span>
      <select v-model="mirrorDirection" :disabled="busy" class="field">
        <option value="pull">{{ $t('modal.mirrorPull') }}</option>
        <option value="push">{{ $t('modal.mirrorPush') }}</option>
      </select>
    </label>
    <p class="text-[85%] text-ink-muted">
      {{ mirrorDirection === 'push' ? $t('modal.mirrorPushHint') : $t('modal.mirrorPullHint') }}
    </p>
    <label class="block text-sm">
      <span class="mb-1 block font-medium">{{ $t('modal.mirrorURL') }}</span>
      <input
        v-model="mirrorURL"
        type="text"
        inputmode="url"
        required
        spellcheck="false"
        autocapitalize="off"
        :disabled="busy"
        class="field font-mono"
        :placeholder="$t('modal.mirrorURLPlaceholder')"
        autofocus
      >
    </label>
    <p v-if="isSSHURL(mirrorURL)" class="text-[85%] text-warn">{{ $t('modal.sshCreateHint') }}</p>
    <label class="block text-sm">
      <span class="mb-1 block font-medium">{{ $t('modal.name') }}</span>
      <input
        v-model="name"
        :disabled="busy"
        class="field"
        required
        maxlength="100"
        :placeholder="$t('modal.repoNamePlaceholder')"
        @input="nameTouched = true"
      >
    </label>
    <label class="block text-sm">
      <span class="mb-1 block font-medium">{{ $t('modal.description') }}</span>
      <textarea
        v-model="description"
        rows="2"
        :disabled="busy"
        class="field min-h-16 resize-y"
        :placeholder="$t('modal.optional')"
      />
    </label>
    <label class="block text-sm">
      <span class="mb-1 block font-medium">{{ $t('modal.mirrorUsername') }}</span>
      <input
        v-model="mirrorUsername"
        :disabled="busy"
        class="field"
        :placeholder="$t('modal.optional')"
        autocomplete="off"
      >
    </label>
    <label class="block text-sm">
      <span class="mb-1 block font-medium">{{ $t('modal.mirrorPassword') }}</span>
      <input
        v-model="mirrorPassword"
        type="password"
        :disabled="busy"
        class="field"
        :placeholder="$t('modal.optional')"
        autocomplete="new-password"
      >
    </label>
    <p class="text-[85%] text-ink-muted">{{ $t('modal.tokenHint') }}</p>
    <p class="text-[85%] text-ink-muted">{{ $t('modal.mirrorLfsUnavailable') }}</p>
    <label v-if="mirrorDirection === 'push'" class="block text-sm">
      <span class="mb-1 block font-medium">{{ $t('modal.defaultBranch') }}</span>
      <input
        v-model="defaultBranch"
        :disabled="busy"
        class="field font-mono"
        :placeholder="$t('modal.defaultBranchPlaceholder')"
      >
    </label>
    <label class="flex items-center gap-2 text-sm">
      <input
        v-model="isPrivate"
        type="checkbox"
        class="accent-ink"
        :disabled="busy"
        @change="isPrivate && (showOnHome = false)"
      >
      {{ $t('modal.private') }}
    </label>
    <label class="flex items-center gap-2 text-sm" :class="isPrivate && 'opacity-50'">
      <input
        v-model="showOnHome"
        type="checkbox"
        class="accent-ink"
        :disabled="busy || isPrivate"
      >
      {{ $t('modal.showOnHome') }}
    </label>
    <p v-if="progressText" class="text-sm text-ink-muted" role="status" aria-live="polite">{{ progressText }}</p>
    <p v-if="error" class="text-sm text-warn">{{ error }}</p>
  </ModalForm>
</template>
