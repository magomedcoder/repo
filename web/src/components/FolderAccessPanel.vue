<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { foldersApi } from '@/api'
import type { Folder, FolderMember, Team } from '@/api/types'
import { localizeError } from '@/i18n'
import { folderKey } from '@/lib/folderNav'
import FolderMembersSection from '@/components/folder/FolderMembersSection.vue'
import FolderTeamsSection from '@/components/folder/FolderTeamsSection.vue'
import FolderTeamDetail from '@/components/folder/FolderTeamDetail.vue'

const props = defineProps<{
  folder: Folder
}>()

const path = computed(() => folderKey(props.folder))
const canManage = computed(() => !!props.folder.caps?.manage_members)

const members = ref<FolderMember[]>([])
const teams = ref<Team[]>([])
const error = ref('')
const loading = ref(false)
const selectedTeam = ref<Team | null>(null)

const membersSectionRef = ref<InstanceType<typeof FolderMembersSection> | null>(null)

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

function onSectionError(message: string) {
  error.value = message
}

function onTeamRemoved(slug: string) {
  if (selectedTeam.value?.slug === slug) {
    selectedTeam.value = null
  }
}

watch(
  () => props.folder.path,
  () => {
    selectedTeam.value = null
    membersSectionRef.value?.applyRolePreset('member')
    load()
  },
  { immediate: true },
)
</script>

<template>
  <div v-if="folder" class="mt-8 space-y-6">
    <p v-if="error" class="text-sm text-warn">{{ error }}</p>
    <p v-if="loading" class="text-sm text-ink-muted">{{ $t('common.loading') }}</p>

    <FolderMembersSection
      ref="membersSectionRef"
      :path="path"
      :can-manage="canManage"
      :members="members"
      @refresh="load"
      @error="onSectionError"
    />

    <FolderTeamsSection
      :path="path"
      :can-manage="canManage"
      :teams="teams"
      @refresh="load"
      @error="onSectionError"
      @select="selectedTeam = $event"
      @removed="onTeamRemoved"
    />

    <FolderTeamDetail
      v-if="selectedTeam"
      :path="path"
      :can-manage="canManage"
      :team="selectedTeam"
      @error="onSectionError"
    />
  </div>
</template>
