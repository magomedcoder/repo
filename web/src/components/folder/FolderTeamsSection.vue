<script setup lang="ts">
import { ref } from 'vue'
import { foldersApi } from '@/api'
import type { Team } from '@/api/types'
import { i18n, localizeError } from '@/i18n'
import Surface from '@/components/ui/Surface.vue'
import SurfaceHeader from '@/components/ui/SurfaceHeader.vue'

const props = defineProps<{
  path: string
  canManage: boolean
  teams: Team[]
}>()

const emit = defineEmits<{
  refresh: []
  error: [message: string]
  select: [team: Team]
  removed: [slug: string]
}>()

const showTeamForm = ref(false)
const teamName = ref('')
const teamBusy = ref(false)

async function createTeam() {
  teamBusy.value = true
  emit('error', '')
  try {
    await foldersApi.createTeam(props.path, { 
      name: teamName.value.trim(), 
      caps: { 
        view: true 
      } 
    })
    teamName.value = ''
    showTeamForm.value = false
    emit('refresh')
  } catch (err) {
    emit('error', localizeError(err, 'errors.saveFailed'))
  } finally {
    teamBusy.value = false
  }
}

async function removeTeam(slug: string) {
  if (!confirm(i18n.global.t('folderAccess.confirmRemoveTeam', { slug }))) {
    return
  }

  emit('error', '')
  try {
    await foldersApi.removeTeam(props.path, slug)
    emit('removed', slug)
    emit('refresh')
  } catch (err) {
    emit('error', localizeError(err, 'errors.deleteFailed'))
  }
}
</script>

<template>
  <Surface>
    <SurfaceHeader class="justify-between">
      <span>{{ $t('folderAccess.teams') }}</span>
      <button
        v-if="canManage"
        type="button"
        class="btn-primary px-2 py-0.5"
        @click="showTeamForm = !showTeamForm"
      >
        {{ $t('folderAccess.addTeam') }}
      </button>
    </SurfaceHeader>
    <form v-if="showTeamForm && canManage" class="space-y-3 border-b border-line p-3 sm:p-4" @submit.prevent="createTeam">
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('folderAccess.teamName') }}</span>
        <input v-model="teamName" class="field" required />
      </label>
      <button type="submit" class="btn-primary disabled:opacity-50" :disabled="teamBusy">
        {{ $t('folderAccess.submitTeam') }}
      </button>
    </form>
    <ul class="divide-y divide-line">
      <li v-for="t in teams" :key="t.slug" class="flex flex-wrap items-center justify-between gap-2 px-3 py-2 sm:px-4">
        <button type="button" class="text-left font-semibold text-ink hover:underline" @click="emit('select', t)">{{ t.name }}</button>
        <button
          v-if="canManage"
          type="button"
          class="btn-warn"
          @click="removeTeam(t.slug)"
        >
          {{ $t('folderAccess.remove') }}
        </button>
      </li>
      <li v-if="!teams.length" class="px-3 py-6 text-center text-sm text-ink-muted">{{ $t('folderAccess.noTeams') }}</li>
    </ul>
  </Surface>
</template>
