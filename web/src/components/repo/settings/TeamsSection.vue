<script setup lang="ts">
import type { RepoCollabRole, RepoTeamGrant, Team } from '@/api/types'
import Icon from '@/components/ui/Icon.vue'
import Surface from '@/components/ui/Surface.vue'
import SurfaceHeader from '@/components/ui/SurfaceHeader.vue'

defineProps<{
  grants: RepoTeamGrant[]
  ownTeams: Team[]
  busy: boolean
  error: string
}>()

const emit = defineEmits<{
  submit: []
  revoke: [slug: string]
}>()

const teamSlug = defineModel<string>('teamSlug', { required: true })
const teamRole = defineModel<RepoCollabRole>('teamRole', { required: true })
const showForm = defineModel<boolean>('showForm', { required: true })
</script>

<template>
  <Surface>
    <SurfaceHeader class="justify-between">
      <span class="inline-flex items-center gap-1.5">
        <Icon name="users" size="sm" class="text-ink-muted" />
        {{ $t('repoSettings.teams.title') }}
      </span>
      <button type="button" class="btn-primary px-2 py-0.5" @click="showForm = !showForm">
        <Icon name="plus" size="sm" />
        {{ $t('repoSettings.teams.grant') }}
      </button>
    </SurfaceHeader>
    <form
      v-if="showForm"
      class="space-y-3 border-b border-line p-3 sm:p-4"
      @submit.prevent="emit('submit')"
    >
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('repoSettings.teams.team') }}</span>
        <select v-model="teamSlug" class="field" required>
          <option disabled value="">{{ $t('repoSettings.teams.pickTeam') }}</option>
          <option v-for="t in ownTeams" :key="t.slug" :value="t.slug">
            {{ t.name }}
          </option>
        </select>
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('repoSettings.teams.role') }}</span>
        <select v-model="teamRole" class="field">
          <option value="read">{{ $t('collaborators.roles.read') }}</option>
          <option value="write">{{ $t('collaborators.roles.write') }}</option>
          <option value="admin">{{ $t('collaborators.roles.admin') }}</option>
        </select>
      </label>
      <button type="submit" class="btn-primary disabled:opacity-50" :disabled="busy || !teamSlug">
        {{ $t('repoSettings.teams.submit') }}
      </button>
    </form>
    <p v-if="error" class="px-3 pt-3 text-sm text-warn">{{ error }}</p>
    <ul class="divide-y divide-line">
      <li
        v-for="g in grants"
        :key="g.team_slug"
        class="flex flex-wrap items-center justify-between gap-2 px-3 py-2 sm:px-4"
      >
        <div>
          <p class="font-semibold">{{ g.team_name }}</p>
          <p class="text-xs text-ink-muted">
            {{ $t(`collaborators.roles.${g.role}`) }}
            <span v-if="!g.owned"> {{ $t('repoSettings.teams.external') }}</span>
          </p>
        </div>
        <button type="button" class="btn-warn" @click="emit('revoke', g.team_slug)">
          <Icon name="trash" size="sm" />
          {{ $t('repoSettings.teams.revoke') }}
        </button>
      </li>
      <li v-if="!grants.length" class="px-3 py-6 text-center text-sm text-ink-muted sm:px-4">
        {{ $t('repoSettings.teams.empty') }}
      </li>
    </ul>
  </Surface>
</template>
