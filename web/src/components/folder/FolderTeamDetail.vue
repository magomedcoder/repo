<script setup lang="ts">
import { ref, watch } from 'vue'
import { foldersApi } from '@/api'
import type { Team, TeamMember, TeamRepoGrant } from '@/api/types'
import { localizeError } from '@/i18n'
import Surface from '@/components/ui/Surface.vue'
import SurfaceHeader from '@/components/ui/SurfaceHeader.vue'

const props = defineProps<{
  path: string
  canManage: boolean
  team: Team
}>()

const emit = defineEmits<{
  error: [message: string]
}>()

const teamMembers = ref<TeamMember[]>([])
const teamRepos = ref<TeamRepoGrant[]>([])
const teamUser = ref('')
const teamRepo = ref('')
const teamRepoRole = ref('read')

async function loadTeamDetail() {
  emit('error', '')
  try {
    const [m, r] = await Promise.all([
      foldersApi.listTeamMembers(props.path, props.team.slug),
      foldersApi.listTeamRepos(props.path, props.team.slug),
    ])
    teamMembers.value = m.members ?? []
    teamRepos.value = r.repos ?? []
  } catch (err) {
    emit('error', localizeError(err, 'errors.loadFailed'))
  }
}

async function addTeamMember() {
  emit('error', '')
  try {
    await foldersApi.addTeamMember(props.path, props.team.slug, { 
      username: teamUser.value.trim() 
    })
    teamUser.value = ''
    await loadTeamDetail()
  } catch (err) {
    emit('error', localizeError(err, 'errors.saveFailed'))
  }
}

async function removeTeamMember(username: string) {
  emit('error', '')
  try {
    await foldersApi.removeTeamMember(props.path, props.team.slug, username)
    await loadTeamDetail()
  } catch (err) {
    emit('error', localizeError(err, 'errors.deleteFailed'))
  }
}

async function setTeamRepo() {
  emit('error', '')
  try {
    await foldersApi.setTeamRepo(props.path, props.team.slug, {
      repo: teamRepo.value.trim(),
      role: teamRepoRole.value,
    })
    teamRepo.value = ''
    await loadTeamDetail()
  } catch (err) {
    emit('error', localizeError(err, 'errors.saveFailed'))
  }
}

async function removeTeamRepo(repo: string) {
  emit('error', '')
  try {
    await foldersApi.removeTeamRepo(props.path, props.team.slug, repo)
    await loadTeamDetail()
  } catch (err) {
    emit('error', localizeError(err, 'errors.deleteFailed'))
  }
}

watch(
  () => props.team.slug,
  () => {
    teamUser.value = ''
    teamRepo.value = ''
    teamRepoRole.value = 'read'
    loadTeamDetail()
  },
  { immediate: true },
)
</script>

<template>
  <Surface>
    <SurfaceHeader>
      {{ $t('folderAccess.teamDetail', { name: team.name }) }}
    </SurfaceHeader>
    <div class="space-y-4 p-3 sm:p-4">
      <div v-if="canManage" class="flex flex-wrap gap-2">
        <input v-model="teamUser" class="field w-auto" :placeholder="$t('folderAccess.username')" />
        <button type="button" class="btn-primary" @click="addTeamMember">
          {{ $t('folderAccess.addTeamMember') }}
        </button>
      </div>
      <Surface as="ul" class="divide-y divide-line">
        <li v-for="m in teamMembers" :key="m.user_id" class="flex items-center justify-between gap-2 px-3 py-2">
          <span>{{ m.username }}</span>
          <button v-if="canManage" type="button" class="text-[10pt] text-warn hover:underline" @click="removeTeamMember(m.username)">{{ $t('folderAccess.remove') }}</button>
        </li>
        <li v-if="!teamMembers.length" class="px-3 py-4 text-center text-sm text-ink-muted">{{ $t('folderAccess.noTeamMembers') }}</li>
      </Surface>

      <div v-if="canManage" class="flex flex-wrap gap-2">
        <input v-model="teamRepo" class="field w-auto" :placeholder="$t('folderAccess.repoName')" />
        <select v-model="teamRepoRole" class="field w-auto">
          <option value="read">{{ $t('collaborators.roles.read') }}</option>
          <option value="write">{{ $t('collaborators.roles.write') }}</option>
          <option value="admin">{{ $t('collaborators.roles.admin') }}</option>
        </select>
        <button type="button" class="btn-primary" @click="setTeamRepo">
          {{ $t('folderAccess.grantRepo') }}
        </button>
      </div>
      <Surface as="ul" class="divide-y divide-line">
        <li v-for="g in teamRepos" :key="g.repo_id" class="flex items-center justify-between gap-2 px-3 py-2">
          <span>{{ g.repo_name }} {{ $t(`collaborators.roles.${g.role}`) }}</span>
          <button v-if="canManage" type="button" class="text-[10pt] text-warn hover:underline" @click="removeTeamRepo(g.repo_name)">{{ $t('folderAccess.remove') }}</button>
        </li>
        <li v-if="!teamRepos.length" class="px-3 py-4 text-center text-sm text-ink-muted">{{ $t('folderAccess.noRepoGrants') }}</li>
      </Surface>
    </div>
  </Surface>
</template>
