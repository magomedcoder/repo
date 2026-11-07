<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { reposApi, teamsApi } from '@/api'
import type { RepoCollabRole, Team, TeamMember, TeamRepoGrant } from '@/api/types'
import Icon from '@/components/ui/Icon.vue'
import PageHeader from '@/components/ui/PageHeader.vue'
import PageState from '@/components/ui/PageState.vue'
import Surface from '@/components/ui/Surface.vue'
import SurfaceHeader from '@/components/ui/SurfaceHeader.vue'
import { repoLocation } from '@/lib/repoNav'
import { i18n, localizeError } from '@/i18n'

const props = defineProps<{
  slug: string
}>()

const router = useRouter()
const team = ref<Team | null>(null)
const members = ref<TeamMember[]>([])
const grants = ref<TeamRepoGrant[]>([])
const ownRepoNames = ref<string[]>([])
const error = ref('')
const loading = ref(true)
const busy = ref(false)

const renameValue = ref('')
const memberUsername = ref('')
const grantRepo = ref('')
const grantRole = ref<RepoCollabRole>('read')

const currentSlug = computed(() => team.value?.slug || props.slug)

async function load() {
  loading.value = true
  error.value = ''
  try {
    team.value = await teamsApi.get(props.slug)
    renameValue.value = team.value.name
    const [memberRes, grantRes] = await Promise.all([
      teamsApi.listMembers(props.slug),
      teamsApi.listRepos(props.slug),
    ])
    members.value = memberRes.members ?? []
    grants.value = grantRes.repos ?? []
    try {
      const own = await reposApi.listOwn()
      ownRepoNames.value = (own.repos ?? []).map((r) => r.name).sort()
    } catch {
      ownRepoNames.value = []
    }
  } catch (err) {
    team.value = null
    error.value = localizeError(err, 'errors.loadFailed')
  } finally {
    loading.value = false
  }
}

async function run(action: () => Promise<unknown>, fallbackKey: Parameters<typeof localizeError>[1]) {
  busy.value = true
  error.value = ''
  try {
    await action()
    return true
  } catch (err) {
    error.value = localizeError(err, fallbackKey)
    return false
  } finally {
    busy.value = false
  }
}

async function rename() {
  const next = renameValue.value.trim()
  if (!team.value || !next || next === team.value.name) {
    return
  }

  const ok = await run(async () => {
    const updated = await teamsApi.rename(currentSlug.value, { name: next })
    if (updated.slug !== props.slug) {
      await router.replace({
        name: 'team',
        params: {
          slug: updated.slug
        }
      })
    }
    team.value = updated
  }, 'errors.saveFailed')
  if (ok) {
    await load()
  }
}

async function removeTeam() {
  if (!team.value || !confirm(i18n.global.t('teams.confirmDelete', { name: team.value.name }))) {
    return
  }

  const ok = await run(() => teamsApi.remove(currentSlug.value), 'errors.deleteFailed')
  if (ok) {
    await router.push({ name: 'teams' })
  }
}

async function addMember() {
  const username = memberUsername.value.trim()
  if (!username) {
    return
  }

  const ok = await run(() => teamsApi.addMember(currentSlug.value, { username }), 'errors.saveFailed')
  if (ok) {
    memberUsername.value = ''
    await load()
  }
}

async function removeMember(username: string) {
  if (!confirm(i18n.global.t('teams.confirmRemoveMember', { username }))) {
    return
  }

  const ok = await run(() => teamsApi.removeMember(currentSlug.value, username), 'errors.deleteFailed')
  if (ok) {
    await load()
  }
}

async function addGrant() {
  const repo = grantRepo.value.trim()
  if (!repo) {
    return
  }

  const ok = await run(() => teamsApi.setRepo(currentSlug.value, { repo, role: grantRole.value }), 'errors.saveFailed')
  if (ok) {
    grantRepo.value = ''
    await load()
  }
}

async function removeGrant(grant: TeamRepoGrant) {
  if (!confirm(i18n.global.t('teams.confirmRevoke', { repo: grant.repo_name }))) {
    return
  }

  const ok = await run(() => teamsApi.removeRepo(currentSlug.value, grant.repo_name), 'errors.deleteFailed')
  if (ok) {
    await load()
  }
}

onMounted(load)
</script>

<template>
  <div class="mx-auto max-w-3xl space-y-4">
    <PageState :loading="loading" :error="team ? '' : error">
      <template v-if="team">
        <PageHeader :title="$t('teams.detail', { name: team.name })" icon="users">
          <template #hint>
            <RouterLink :to="{ name: 'teams' }" class="text-link">{{ $t('teams.backToList') }}</RouterLink>
          </template>
          <template #actions>
            <button type="button" class="btn-warn" :disabled="busy" @click="removeTeam">
              <Icon name="trash" size="sm" />
              {{ $t('teams.delete') }}
            </button>
          </template>
        </PageHeader>

        <p v-if="error" class="text-sm text-warn">{{ error }}</p>

        <Surface as="form" @submit.prevent="rename">
          <SurfaceHeader>{{ $t('teams.rename') }}</SurfaceHeader>
          <div class="flex flex-wrap items-end gap-3 p-3 sm:p-4">
            <label class="block min-w-48 flex-1 text-sm">
              <span class="mb-1 block font-semibold">{{ $t('teams.name') }}</span>
              <input v-model="renameValue" class="field" required maxlength="64">
            </label>
            <button type="submit" class="btn-primary disabled:opacity-50" :disabled="busy">
              {{ $t('teams.save') }}
            </button>
          </div>
        </Surface>

        <Surface>
          <SurfaceHeader>
            <span class="inline-flex items-center gap-1.5">
              <Icon name="users" size="sm" class="text-ink-muted" />
              {{ $t('teams.members') }}
            </span>
          </SurfaceHeader>
          <form class="flex flex-wrap items-end gap-3 border-b border-line p-3 sm:p-4" @submit.prevent="addMember">
            <label class="block min-w-48 flex-1 text-sm">
              <span class="mb-1 block font-semibold">{{ $t('teams.username') }}</span>
              <input v-model="memberUsername" class="field" required>
            </label>
            <button type="submit" class="btn-primary disabled:opacity-50" :disabled="busy">
              {{ $t('teams.addMember') }}
            </button>
          </form>
          <ul class="divide-y divide-line">
            <li
              v-for="m in members"
              :key="m.user_id"
              class="flex flex-wrap items-center justify-between gap-2 px-3 py-2 sm:px-4"
            >
              <span class="font-semibold">{{ m.username }}</span>
              <button type="button" class="btn-warn" :disabled="busy" @click="removeMember(m.username)">
                <Icon name="trash" size="sm" />
                {{ $t('teams.remove') }}
              </button>
            </li>
            <li v-if="!members.length" class="px-3 py-6 text-center text-sm text-ink-muted sm:px-4">
              {{ $t('teams.noMembers') }}
            </li>
          </ul>
        </Surface>

        <Surface>
          <SurfaceHeader>
            <span class="inline-flex items-center gap-1.5">
              <Icon name="repo" size="sm" class="text-ink-muted" />
              {{ $t('teams.repos') }}
            </span>
          </SurfaceHeader>
          <form class="flex flex-wrap items-end gap-3 border-b border-line p-3 sm:p-4" @submit.prevent="addGrant">
            <label class="block min-w-48 flex-1 text-sm">
              <span class="mb-1 block font-semibold">{{ $t('teams.repoName') }}</span>
              <input
                v-model="grantRepo"
                class="field"
                list="team-own-repos"
                required
              >
              <datalist id="team-own-repos">
                <option v-for="name in ownRepoNames" :key="name" :value="name" />
              </datalist>
            </label>
            <label class="block text-sm">
              <span class="mb-1 block font-semibold">{{ $t('collaborators.role') }}</span>
              <select v-model="grantRole" class="field">
                <option value="read">{{ $t('collaborators.roles.read') }}</option>
                <option value="write">{{ $t('collaborators.roles.write') }}</option>
                <option value="admin">{{ $t('collaborators.roles.admin') }}</option>
              </select>
            </label>
            <button type="submit" class="btn-primary disabled:opacity-50" :disabled="busy">
              {{ $t('teams.grant') }}
            </button>
          </form>
          <ul class="divide-y divide-line">
            <li
              v-for="g in grants"
              :key="g.repo_id"
              class="flex flex-wrap items-center justify-between gap-2 px-3 py-2 sm:px-4"
            >
              <div class="min-w-0">
                <RouterLink :to="repoLocation('repo', g.repo_path || g.repo_name)" class="font-semibold">
                  {{ g.repo_name }}
                </RouterLink>
                <p class="text-xs text-ink-muted">{{ $t(`collaborators.roles.${g.role}`) }}</p>
              </div>
              <button type="button" class="btn-warn" :disabled="busy" @click="removeGrant(g)">
                <Icon name="trash" size="sm" />
                {{ $t('teams.revoke') }}
              </button>
            </li>
            <li v-if="!grants.length" class="px-3 py-6 text-center text-sm text-ink-muted sm:px-4">
              {{ $t('teams.noGrants') }}
            </li>
          </ul>
        </Surface>
      </template>
    </PageState>
  </div>
</template>
