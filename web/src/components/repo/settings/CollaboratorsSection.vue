<script setup lang="ts">
import type { RepoCollaborator, RepoCollabRole } from '@/api/types'
import Icon from '@/components/ui/Icon.vue'
import Surface from '@/components/ui/Surface.vue'
import SurfaceHeader from '@/components/ui/SurfaceHeader.vue'

defineProps<{
  collaborators: RepoCollaborator[]
  collabBusy: boolean
  collabError: string
}>()

const emit = defineEmits<{
  submit: []
  remove: [username: string]
}>()

const collabUsername = defineModel<string>('collabUsername', { required: true })
const collabRole = defineModel<RepoCollabRole>('collabRole', { required: true })
const showCollabForm = defineModel<boolean>('showCollabForm', { required: true })
</script>

<template>
  <Surface>
    <SurfaceHeader class="justify-between">
      <span class="inline-flex items-center gap-1.5">
        <Icon name="users" size="sm" class="text-ink-muted" />
        {{ $t('collaborators.title') }}
      </span>
      <button type="button" class="btn-primary px-2 py-0.5" @click="showCollabForm = !showCollabForm">
        <Icon name="plus" size="sm" />
        {{ $t('collaborators.add') }}
      </button>
    </SurfaceHeader>
    <form
      v-if="showCollabForm"
      class="space-y-3 border-b border-line p-3 sm:p-4"
      @submit.prevent="emit('submit')"
    >
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('collaborators.username') }}</span>
        <input v-model="collabUsername" class="field" required />
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('collaborators.role') }}</span>
        <select v-model="collabRole" class="field">
          <option value="read">{{ $t('collaborators.roles.read') }}</option>
          <option value="write">{{ $t('collaborators.roles.write') }}</option>
          <option value="admin">{{ $t('collaborators.roles.admin') }}</option>
        </select>
      </label>
      <button type="submit" class="btn-primary disabled:opacity-50" :disabled="collabBusy">
        {{ $t('collaborators.submit') }}
      </button>
    </form>
    <p v-if="collabError" class="px-3 pt-3 text-sm text-warn">{{ collabError }}</p>
    <ul class="divide-y divide-line">
      <li
        v-for="c in collaborators"
        :key="c.user_id"
        class="flex flex-wrap items-center justify-between gap-2 px-3 py-2 sm:px-4"
      >
        <div>
          <p class="font-semibold">{{ c.username }}</p>
          <p class="text-xs text-ink-muted">{{ $t(`collaborators.roles.${c.role}`) }}</p>
        </div>
        <button type="button" class="btn-warn" @click="emit('remove', c.username)">
          <Icon name="trash" size="sm" />
          {{ $t('collaborators.remove') }}
        </button>
      </li>
      <li v-if="!collaborators.length" class="px-3 py-6 text-center text-sm text-ink-muted sm:px-4">
        {{ $t('collaborators.empty') }}
      </li>
    </ul>
  </Surface>
</template>
