<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { reposApi, teamsApi } from '@/api'
import type { RefInfo, RepoCollaborator, RepoCollabRole, RepoTeamGrant, Team } from '@/api/types'
import MirrorSettingsForm from '@/components/repo/MirrorSettingsForm.vue'
import CollaboratorsSection from '@/components/repo/settings/CollaboratorsSection.vue'
import DangerZone from '@/components/repo/settings/DangerZone.vue'
import GeneralSettingsForm from '@/components/repo/settings/GeneralSettingsForm.vue'
import SettingsNav from '@/components/repo/settings/SettingsNav.vue'
import TeamsSection from '@/components/repo/settings/TeamsSection.vue'
import { SETTINGS_SECTIONS, type SettingsSection } from '@/components/repo/settings/types'
import RepoShell from '@/components/repo/RepoShell.vue'
import PageState from '@/components/ui/PageState.vue'
import { useCanWrite } from '@/composables/useCanWrite'
import { invalidateRepo, useRepo } from '@/composables/useRepo'
import type { PageError } from '@/i18n'
import { i18n, localizeError } from '@/i18n'
import { repositoryPath } from '@/lib/repoPath'
import { repoLocation } from '@/lib/repoNav'

const props = defineProps<{
  repoPath: string
}>()

const route = useRoute()
const router = useRouter()
const { repo, loading: repoLoading, error: repoError, reload } = useRepo(() => props.repoPath)
const { canAdmin } = useCanWrite(repo)

const section = computed<SettingsSection>(() => {
  const id = route.hash.replace(/^#/, '') as SettingsSection
  return SETTINGS_SECTIONS.some((s) => s.id === id) ? id : 'general'
})

function settingsLocation(path: string, sectionId: SettingsSection = section.value) {
  const loc = repoLocation('repo-settings', path)
  const base = typeof loc === 'string' ? { path: loc } : { ...loc }
  return { ...base, hash: `#${sectionId}` }
}

const collaborators = ref<RepoCollaborator[]>([])
const collabUsername = ref('')
const collabRole = ref<RepoCollabRole>('write')
const collabBusy = ref(false)
const collabError = ref('')
const showCollabForm = ref(false)

const teamGrants = ref<RepoTeamGrant[]>([])
const ownTeams = ref<Team[]>([])
const teamSlug = ref('')
const teamRole = ref<RepoCollabRole>('read')
const teamBusy = ref(false)
const teamError = ref('')
const showTeamForm = ref(false)

const name = ref('')
const description = ref('')
const isPrivate = ref(false)
const showOnHome = ref(false)
const protectBranch = ref(true)
const issuesEnabled = ref(true)
const defaultBranch = ref('main')
const branches = ref<RefInfo[]>([])
const loadError = ref<PageError | string>('')
const formError = ref('')
const message = ref('')
const dataLoading = ref(false)
const saving = ref(false)
const deleting = ref(false)
const redirected = ref(false)

const loading = computed(() => dataLoading.value)

const branchOptions = computed(() => {
  const names = branches.value.map((b) => b.name)
  if (defaultBranch.value && !names.includes(defaultBranch.value)) {
    return [defaultBranch.value, ...names]
  }

  return names
})

function syncFormFromRepo() {
  if (!repo.value) {
    return
  }

  name.value = repo.value.name
  description.value = repo.value.description
  isPrivate.value = repo.value.is_private
  showOnHome.value = !!repo.value.show_on_home && !repo.value.is_private
  protectBranch.value = repo.value.protect_default_branch !== false
  issuesEnabled.value = repo.value.issues_enabled !== false
  defaultBranch.value = repo.value.default_branch
}

async function denyNonOwner() {
  if (redirected.value) {
    return
  }

  redirected.value = true
  await router.replace(repoLocation('repo', props.repoPath))
}

async function load() {
  if (!repo.value) {
    return
  }

  if (!canAdmin.value) {
    await denyNonOwner()
    return
  }

  dataLoading.value = true
  loadError.value = ''
  message.value = ''
  collabError.value = ''
  teamError.value = ''
  try {
    syncFormFromRepo()
    const [branchRes, collabRes, teamRes, ownRes] = await Promise.all([
      reposApi.branches(props.repoPath),
      reposApi.listCollaborators(props.repoPath),
      reposApi.listTeams(props.repoPath),
      teamsApi.list().catch(() => ({ teams: [] as Team[] })),
    ])
    collaborators.value = collabRes.collaborators ?? []
    teamGrants.value = teamRes.teams ?? []
    ownTeams.value = ownRes.teams ?? []
    branches.value = branchRes.branches ?? []
    if (branches.value.length && !branches.value.some((b) => b.name === defaultBranch.value)) {
      defaultBranch.value = branches.value[0]?.name || defaultBranch.value
    }
  } catch (err) {
    loadError.value = localizeError(err, 'errors.loadSettings')
  } finally {
    dataLoading.value = false
  }
}

async function save() {
  if (!repo.value || !canAdmin.value) {
    return
  }

  saving.value = true
  formError.value = ''
  message.value = ''
  try {
    const nextName = name.value.trim()
    const updated = await reposApi.update(props.repoPath, {
      name: nextName,
      description: description.value,
      private: isPrivate.value,
      show_on_home: !isPrivate.value && showOnHome.value,
      default_branch: defaultBranch.value,
      protect_default_branch: protectBranch.value,
      issues_enabled: issuesEnabled.value,
    })

    const currentPath = repositoryPath(updated)

    if (currentPath !== props.repoPath) {
      invalidateRepo(props.repoPath)
      invalidateRepo(currentPath)
      await router.replace(settingsLocation(currentPath))
      message.value = i18n.global.t('settings.saved')
      return
    }

    message.value = i18n.global.t('settings.saved')
    invalidateRepo(props.repoPath)
    await reload()
  } catch (err) {
    formError.value = localizeError(err, 'errors.saveFailed')
  } finally {
    saving.value = false
  }
}

async function remove() {
  if (!canAdmin.value) {
    return
  }

  if (!confirm(i18n.global.t('settings.confirmDelete', { path: props.repoPath }))) {
    return
  }

  const repoName = repo.value?.name || name.value.trim()
  const typed = window.prompt(i18n.global.t('settings.confirmDeleteName', { name: repoName }), '')
  if (typed !== repoName) {
    return
  }

  const password = window.prompt(i18n.global.t('settings.confirmDeletePassword'), '')
  if (!password) {
    return
  }

  deleting.value = true
  formError.value = ''
  try {
    await reposApi.remove(props.repoPath, repoName, password)
    await router.push({ name: 'home' })
  } catch (err) {
    formError.value = localizeError(err, 'errors.deleteFailed')
    deleting.value = false
  }
}

async function addCollaborator() {
  collabBusy.value = true
  collabError.value = ''

  try {
    await reposApi.addCollaborator(props.repoPath, {
      username: collabUsername.value.trim(),
      role: collabRole.value,
    })
    collabUsername.value = ''
    showCollabForm.value = false
    const res = await reposApi.listCollaborators(props.repoPath)
    collaborators.value = res.collaborators ?? []
  } catch (err) {
    collabError.value = localizeError(err, 'errors.saveFailed')
  } finally {
    collabBusy.value = false
  }
}

async function removeCollaborator(username: string) {
  if (!confirm(i18n.global.t('collaborators.confirmRemove', { username }))) {
    return
  }

  collabError.value = ''
  try {
    await reposApi.removeCollaborator(props.repoPath, username)
    collaborators.value = collaborators.value.filter((c) => c.username !== username)
  } catch (err) {
    collabError.value = localizeError(err, 'errors.deleteFailed')
  }
}

async function grantTeam() {
  teamBusy.value = true
  teamError.value = ''
  try {
    await reposApi.grantTeam(props.repoPath, {
      team: teamSlug.value,
      role: teamRole.value,
    })
    teamSlug.value = ''
    showTeamForm.value = false
    const res = await reposApi.listTeams(props.repoPath)
    teamGrants.value = res.teams ?? []
  } catch (err) {
    teamError.value = localizeError(err, 'errors.saveFailed')
  } finally {
    teamBusy.value = false
  }
}

async function revokeTeam(slug: string) {
  const grant = teamGrants.value.find((g) => g.team_slug === slug)
  if (!confirm(i18n.global.t('repoSettings.teams.confirmRevoke', { team: grant?.team_name || slug }))) {
    return
  }

  teamError.value = ''
  try {
    await reposApi.revokeTeam(props.repoPath, slug)
    teamGrants.value = teamGrants.value.filter((g) => g.team_slug !== slug)
  } catch (err) {
    teamError.value = localizeError(err, 'errors.deleteFailed')
  }
}

watch(
  () => props.repoPath,
  () => {
    redirected.value = false
  },
)

watch([() => props.repoPath, repo, canAdmin], load, { immediate: true })
</script>

<template>
  <RepoShell
    :repo-path="repoPath"
    tab="settings"
    :repo="repo"
    :loading="repoLoading"
    :error="repoError"
  >
    <PageState v-if="canAdmin" :loading="loading" :error="loadError">
      <div class="flex flex-col gap-4 md:flex-row md:items-start md:gap-6">
        <SettingsNav :repo-path="repoPath" :section="section" />

        <div class="min-w-0 flex-1">
          <GeneralSettingsForm
            v-if="section === 'general'"
            v-model:name="name"
            v-model:description="description"
            v-model:default-branch="defaultBranch"
            v-model:is-private="isPrivate"
            v-model:show-on-home="showOnHome"
            v-model:protect-branch="protectBranch"
            v-model:issues-enabled="issuesEnabled"
            :branch-options="branchOptions"
            :form-error="formError"
            :message="message"
            :saving="saving"
            @submit="save"
          />

          <MirrorSettingsForm
            v-else-if="section === 'mirror' && repo"
            :key="repoPath"
            :repo-path="repoPath"
            :repo="repo"
            @changed="invalidateRepo(repoPath)"
          />

          <CollaboratorsSection
            v-else-if="section === 'collaborators'"
            v-model:collab-username="collabUsername"
            v-model:collab-role="collabRole"
            v-model:show-collab-form="showCollabForm"
            :collaborators="collaborators"
            :collab-busy="collabBusy"
            :collab-error="collabError"
            @submit="addCollaborator"
            @remove="removeCollaborator"
          />

          <TeamsSection
            v-else-if="section === 'teams'"
            v-model:team-slug="teamSlug"
            v-model:team-role="teamRole"
            v-model:show-form="showTeamForm"
            :grants="teamGrants"
            :own-teams="ownTeams"
            :busy="teamBusy"
            :error="teamError"
            @submit="grantTeam"
            @revoke="revokeTeam"
          />

          <DangerZone
            v-else-if="section === 'danger'"
            :deleting="deleting"
            @delete="remove"
          />
        </div>
      </div>
    </PageState>
  </RepoShell>
</template>
