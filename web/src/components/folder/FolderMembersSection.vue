<script setup lang="ts">
import { ref } from 'vue'
import { foldersApi } from '@/api'
import type { FolderCaps, FolderMember, FolderRole } from '@/api/types'
import { i18n, localizeError } from '@/i18n'
import Surface from '@/components/ui/Surface.vue'
import SurfaceHeader from '@/components/ui/SurfaceHeader.vue'

const props = defineProps<{
  path: string
  canManage: boolean
  members: FolderMember[]
}>()

const emit = defineEmits<{
  refresh: []
  error: [message: string]
}>()

const showMemberForm = ref(false)
const editingMember = ref<FolderMember | null>(null)
const memberUsername = ref('')
const memberRole = ref<FolderRole>('member')
const memberCaps = ref<FolderCaps>({ view: true })
const memberBusy = ref(false)

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

function resetMemberForm() {
  showMemberForm.value = false
  editingMember.value = null
  memberUsername.value = ''
  applyRolePreset('member')
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

async function saveMember() {
  memberBusy.value = true
  emit('error', '')
  try {
    if (editingMember.value) {
      await foldersApi.updateMember(props.path, editingMember.value.username, {
        role: memberRole.value,
        caps: memberCaps.value,
      })
    } else {
      await foldersApi.addMember(props.path, {
        username: memberUsername.value.trim(),
        role: memberRole.value,
        caps: memberCaps.value,
      })
    }
    resetMemberForm()
    emit('refresh')
  } catch (err) {
    emit('error', localizeError(err, 'errors.saveFailed'))
  } finally {
    memberBusy.value = false
  }
}

async function removeMember(username: string) {
  if (!confirm(i18n.global.t('folderAccess.confirmRemoveMember', { username }))) {
    return
  }

  emit('error', '')
  try {
    await foldersApi.removeMember(props.path, username)
    emit('refresh')
  } catch (err) {
    emit('error', localizeError(err, 'errors.deleteFailed'))
  }
}

defineExpose({ resetMemberForm, applyRolePreset })
</script>

<template>
  <Surface>
    <SurfaceHeader class="justify-between">
      <span>{{ $t('folderAccess.members') }}</span>
      <button
        v-if="canManage"
        type="button"
        class="btn-primary px-2 py-0.5"
        @click="openAddMember"
      >
        {{ $t('folderAccess.addMember') }}
      </button>
    </SurfaceHeader>
    <form v-if="showMemberForm && canManage" class="space-y-3 border-b border-line p-3 sm:p-4" @submit.prevent="saveMember">
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('folderAccess.username') }}</span>
        <input
          v-model="memberUsername"
          class="field disabled:opacity-60"
          :disabled="!!editingMember"
          required
        />
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('folderAccess.role') }}</span>
        <select class="field" :value="memberRole" @change="applyRolePreset(($event.target as HTMLSelectElement).value as FolderRole)">
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
        <button type="submit" class="btn-primary disabled:opacity-50" :disabled="memberBusy">
          {{ editingMember ? $t('folderAccess.saveMember') : $t('folderAccess.submitMember') }}
        </button>
        <button type="button" class="btn" @click="resetMemberForm">
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
            class="btn"
            @click="openEditMember(m)"
          >
            {{ $t('folderAccess.edit') }}
          </button>
          <button
            type="button"
            class="btn-warn"
            @click="removeMember(m.username)"
          >
            {{ $t('folderAccess.remove') }}
          </button>
        </div>
      </li>
      <li v-if="!members.length" class="px-3 py-6 text-center text-sm text-ink-muted">{{ $t('folderAccess.noMembers') }}</li>
    </ul>
  </Surface>
</template>
