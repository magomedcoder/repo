<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { foldersApi } from '@/api'
import type { Folder, FolderCaps, FolderMember, FolderRole, Team, TeamMember, TeamRepoGrant } from '@/api/types'
import { i18n, localizeError } from '@/i18n'
import { folderKey } from '@/lib/folderNav'

const props = defineProps<{
  folder: Folder
}>()

const path = computed(() => folderKey(props.folder))
const canManage = computed(() => !!props.folder.caps?.manage_members)

const members = ref<FolderMember[]>([])
const teams = ref<Team[]>([])
const error = ref('')
const loading = ref(false)

const showMemberForm = ref(false)
const editingMember = ref<FolderMember | null>(null)
const memberUsername = ref('')
const memberRole = ref<FolderRole>('member')
const memberCaps = ref<FolderCaps>({ view: true })
const memberBusy = ref(false)

const showTeamForm = ref(false)
const teamName = ref('')
const teamBusy = ref(false)
const selectedTeam = ref<Team | null>(null)
const teamMembers = ref<TeamMember[]>([])
const teamRepos = ref<TeamRepoGrant[]>([])
const teamUser = ref('')
const teamRepo = ref('')
const teamRepoRole = ref('read')

const capKeys: (keyof FolderCaps)[] = [
  'view',
  'create_repos',
  'manage_members',
  'manage_settings',
  'delete_folder',
  'delete_repos',
]

function applyRolePreset(role: FolderRole) {
  memberRole.value = role
  if (role === 'owner') {
    memberCaps.value = Object.fromEntries(capKeys.map((k) => [k, true])) as FolderCaps
  } else if (role === 'admin') {
    memberCaps.value = Object.fromEntries(capKeys.map((k) => [k, k !== 'delete_folder'])) as FolderCaps
  } else {
    memberCaps.value = { view: true }
  }
}

async function load() {
  if (!path.value) {
    return
  }

  loading.value = true
  error.value = ''
  try {
    const [m, t] = await Promise.all([
      foldersApi.listMembers(path.value),
      foldersApi.listTeams(path.value),
    ])
    members.value = m.members ?? []
    teams.value = t.teams ?? []
  } catch (err) {
    error.value = localizeError(err, 'errors.loadFailed')
  } finally {
    loading.value = false
  }
}

function openAddMember() {
  editingMember.value = null
  memberUsername.value = ''
  applyRolePreset('member')
  showMemberForm.value = !showMemberForm.value
}

function asFolderRole(role: string): FolderRole {
  if (role === 'owner' || role === 'admin' || role === 'member') {
    return role
  }

  return 'member'
}

function openEditMember(m: FolderMember) {
  editingMember.value = m
  memberUsername.value = m.username
  memberRole.value = asFolderRole(m.role)
  memberCaps.value = { ...m.caps }
  showMemberForm.value = true
}

function closeMemberForm() {
  showMemberForm.value = false
  editingMember.value = null
  memberUsername.value = ''
  applyRolePreset('member')
}

async function saveMember() {
  memberBusy.value = true
  error.value = ''
  try {
    if (editingMember.value) {
      await foldersApi.updateMember(path.value, editingMember.value.username, {
        role: memberRole.value,
        caps: memberCaps.value,
      })
    } else {
      await foldersApi.addMember(path.value, {
        username: memberUsername.value.trim(),
        role: memberRole.value,
        caps: memberCaps.value,
      })
    }
    closeMemberForm()
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.saveFailed')
  } finally {
    memberBusy.value = false
  }
}

async function removeMember(username: string) {
  if (!confirm(i18n.global.t('folderAccess.confirmRemoveMember', { username }))) {
    return
  }

  error.value = ''
  try {
    await foldersApi.removeMember(path.value, username)
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.deleteFailed')
  }
}

async function createTeam() {
  teamBusy.value = true
  error.value = ''
  try {
    await foldersApi.createTeam(path.value, { name: teamName.value.trim(), caps: { view: true } })
    teamName.value = ''
    showTeamForm.value = false
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.saveFailed')
  } finally {
    teamBusy.value = false
  }
}

async function removeTeam(slug: string) {
  if (!confirm(i18n.global.t('folderAccess.confirmRemoveTeam', { slug }))) {
    return
  }

  error.value = ''
  try {
    await foldersApi.removeTeam(path.value, slug)
    if (selectedTeam.value?.slug === slug) {
      selectedTeam.value = null
    }
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.deleteFailed')
  }
}

async function openTeam(team: Team) {
  selectedTeam.value = team
  error.value = ''
  try {
    const [m, r] = await Promise.all([
      foldersApi.listTeamMembers(path.value, team.slug),
      foldersApi.listTeamRepos(path.value, team.slug),
    ])
    teamMembers.value = m.members ?? []
    teamRepos.value = r.repos ?? []
  } catch (err) {
    error.value = localizeError(err, 'errors.loadFailed')
  }
}

async function addTeamMember() {
  if (!selectedTeam.value) {
    return
  }

  error.value = ''
  try {
    await foldersApi.addTeamMember(path.value, selectedTeam.value.slug, { username: teamUser.value.trim() })
    teamUser.value = ''
    await openTeam(selectedTeam.value)
  } catch (err) {
    error.value = localizeError(err, 'errors.saveFailed')
  }
}

async function removeTeamMember(username: string) {
  if (!selectedTeam.value) {
    return
  }

  error.value = ''
  try {
    await foldersApi.removeTeamMember(path.value, selectedTeam.value.slug, username)
    await openTeam(selectedTeam.value)
  } catch (err) {
    error.value = localizeError(err, 'errors.deleteFailed')
  }
}

async function setTeamRepo() {
  if (!selectedTeam.value) {
    return
  }

  error.value = ''
  try {
    await foldersApi.setTeamRepo(path.value, selectedTeam.value.slug, {
      repo: teamRepo.value.trim(),
      role: teamRepoRole.value,
    })
    teamRepo.value = ''
    await openTeam(selectedTeam.value)
  } catch (err) {
    error.value = localizeError(err, 'errors.saveFailed')
  }
}

async function removeTeamRepo(repo: string) {
  if (!selectedTeam.value) {
    return
  }

  error.value = ''
  try {
    await foldersApi.removeTeamRepo(path.value, selectedTeam.value.slug, repo)
    await openTeam(selectedTeam.value)
  } catch (err) {
    error.value = localizeError(err, 'errors.deleteFailed')
  }
}

watch(
  () => props.folder.path,
  () => {
    selectedTeam.value = null
    applyRolePreset('member')
    load()
  },
  { immediate: true },
)
</script>

<template>
  <div v-if="folder" class="mt-8 space-y-6">
    <p v-if="error" class="text-sm text-warn">{{ error }}</p>
    <p v-if="loading" class="text-sm text-ink-muted">{{ $t('common.loading') }}</p>

    <div class="overflow-hidden border border-line bg-paper">
      <div class="flex items-center justify-between gap-2 border-b border-line bg-path px-3 py-1.5 text-[10pt] text-ink">
        <span>{{ $t('folderAccess.members') }}</span>
        <button
          v-if="canManage"
          type="button"
          class="inline-flex items-center justify-center border border-ink bg-ink px-2 py-0.5 text-[10pt] text-paper hover:bg-ink-muted"
          @click="openAddMember"
        >
          {{ $t('folderAccess.addMember') }}
        </button>
      </div>
      <form v-if="showMemberForm && canManage" class="space-y-3 border-b border-line p-3 sm:p-4" @submit.prevent="saveMember">
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('folderAccess.username') }}</span>
          <input
            v-model="memberUsername"
            class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none disabled:opacity-60"
            :disabled="!!editingMember"
            required
          />
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('folderAccess.role') }}</span>
          <select class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none" :value="memberRole" @change="applyRolePreset(($event.target as HTMLSelectElement).value as FolderRole)">
            <option value="member">{{ $t('folderAccess.roles.member') }}</option>
            <option value="admin">{{ $t('folderAccess.roles.admin') }}</option>
            <option value="owner">{{ $t('folderAccess.roles.owner') }}</option>
          </select>
        </label>
        <div class="grid gap-2 sm:grid-cols-2">
          <label v-for="key in capKeys" :key="key" class="flex items-center gap-2 text-sm">
            <input v-model="memberCaps[key]" type="checkbox" class="accent-ink" :disabled="key === 'view' || memberRole === 'owner'" />
            {{ $t(`folderAccess.caps.${key}`) }}
          </label>
        </div>
        <div class="flex flex-wrap gap-2">
          <button type="submit" class="inline-flex items-center justify-center border border-ink bg-ink px-2.5 py-1 text-[10pt] text-paper hover:bg-ink-muted disabled:opacity-50" :disabled="memberBusy">
            {{ editingMember ? $t('folderAccess.saveMember') : $t('folderAccess.submitMember') }}
          </button>
          <button type="button" class="inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-ink hover:bg-path" @click="closeMemberForm">
            {{ $t('common.cancel') }}
          </button>
        </div>
      </form>
      <ul class="divide-y divide-line">
        <li v-for="m in members" :key="m.user_id" class="flex flex-wrap items-center justify-between gap-2 px-3 py-2 sm:px-4">
          <div>
            <p class="font-semibold">{{ m.username }}</p>
            <p class="text-xs text-ink-muted">{{ $t(`folderAccess.roles.${m.role}`) }}</p>
          </div>
          <div v-if="canManage" class="flex flex-wrap gap-2">
            <button
              type="button"
              class="inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-ink hover:bg-path"
              @click="openEditMember(m)"
            >
              {{ $t('folderAccess.edit') }}
            </button>
            <button
              type="button"
              class="inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-warn hover:bg-warn-soft"
              @click="removeMember(m.username)"
            >
              {{ $t('folderAccess.remove') }}
            </button>
          </div>
        </li>
        <li v-if="!members.length" class="px-3 py-6 text-center text-sm text-ink-muted">{{ $t('folderAccess.noMembers') }}</li>
      </ul>
    </div>

    <div class="overflow-hidden border border-line bg-paper">
      <div class="flex items-center justify-between gap-2 border-b border-line bg-path px-3 py-1.5 text-[10pt] text-ink">
        <span>{{ $t('folderAccess.teams') }}</span>
        <button
          v-if="canManage"
          type="button"
          class="inline-flex items-center justify-center border border-ink bg-ink px-2 py-0.5 text-[10pt] text-paper hover:bg-ink-muted"
          @click="showTeamForm = !showTeamForm"
        >
          {{ $t('folderAccess.addTeam') }}
        </button>
      </div>
      <form v-if="showTeamForm && canManage" class="space-y-3 border-b border-line p-3 sm:p-4" @submit.prevent="createTeam">
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('folderAccess.teamName') }}</span>
          <input v-model="teamName" class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none" required />
        </label>
        <button type="submit" class="inline-flex items-center justify-center border border-ink bg-ink px-2.5 py-1 text-[10pt] text-paper hover:bg-ink-muted disabled:opacity-50" :disabled="teamBusy">
          {{ $t('folderAccess.submitTeam') }}
        </button>
      </form>
      <ul class="divide-y divide-line">
        <li v-for="t in teams" :key="t.slug" class="flex flex-wrap items-center justify-between gap-2 px-3 py-2 sm:px-4">
          <button type="button" class="text-left font-semibold text-ink hover:underline" @click="openTeam(t)">{{ t.name }}</button>
          <button
            v-if="canManage"
            type="button"
            class="inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-warn hover:bg-warn-soft"
            @click="removeTeam(t.slug)"
          >
            {{ $t('folderAccess.remove') }}
          </button>
        </li>
        <li v-if="!teams.length" class="px-3 py-6 text-center text-sm text-ink-muted">{{ $t('folderAccess.noTeams') }}</li>
      </ul>
    </div>

    <div v-if="selectedTeam" class="overflow-hidden border border-line bg-paper">
      <div class="border-b border-line bg-path px-3 py-1.5 text-[10pt] text-ink">
        {{ $t('folderAccess.teamDetail', { name: selectedTeam.name }) }}
      </div>
      <div class="space-y-4 p-3 sm:p-4">
        <div v-if="canManage" class="flex flex-wrap gap-2">
          <input v-model="teamUser" class="border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none" :placeholder="$t('folderAccess.username')" />
          <button type="button" class="inline-flex items-center justify-center border border-ink bg-ink px-2.5 py-1 text-[10pt] text-paper hover:bg-ink-muted" @click="addTeamMember">
            {{ $t('folderAccess.addTeamMember') }}
          </button>
        </div>
        <ul class="divide-y divide-line border border-line">
          <li v-for="m in teamMembers" :key="m.user_id" class="flex items-center justify-between gap-2 px-3 py-2">
            <span>{{ m.username }}</span>
            <button v-if="canManage" type="button" class="text-[10pt] text-warn hover:underline" @click="removeTeamMember(m.username)">{{ $t('folderAccess.remove') }}</button>
          </li>
          <li v-if="!teamMembers.length" class="px-3 py-4 text-center text-sm text-ink-muted">{{ $t('folderAccess.noTeamMembers') }}</li>
        </ul>

        <div v-if="canManage" class="flex flex-wrap gap-2">
          <input v-model="teamRepo" class="border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none" :placeholder="$t('folderAccess.repoName')" />
          <select v-model="teamRepoRole" class="border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none">
            <option value="read">{{ $t('collaborators.roles.read') }}</option>
            <option value="write">{{ $t('collaborators.roles.write') }}</option>
            <option value="admin">{{ $t('collaborators.roles.admin') }}</option>
          </select>
          <button type="button" class="inline-flex items-center justify-center border border-ink bg-ink px-2.5 py-1 text-[10pt] text-paper hover:bg-ink-muted" @click="setTeamRepo">
            {{ $t('folderAccess.grantRepo') }}
          </button>
        </div>
        <ul class="divide-y divide-line border border-line">
          <li v-for="g in teamRepos" :key="g.repo_id" class="flex items-center justify-between gap-2 px-3 py-2">
            <span>{{ g.repo_name }} {{ $t(`collaborators.roles.${g.role}`) }}</span>
            <button v-if="canManage" type="button" class="text-[10pt] text-warn hover:underline" @click="removeTeamRepo(g.repo_name)">{{ $t('folderAccess.remove') }}</button>
          </li>
          <li v-if="!teamRepos.length" class="px-3 py-4 text-center text-sm text-ink-muted">{{ $t('folderAccess.noRepoGrants') }}</li>
        </ul>
      </div>
    </div>
  </div>
</template>
