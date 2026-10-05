<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { orgsApi } from '@/api'
import type { OrgMember, Organization, Repository } from '@/api/types'
import CreateRepoModal from '@/components/CreateRepoModal.vue'
import PageState from '@/components/ui/PageState.vue'
import VisibilityBadge from '@/components/ui/VisibilityBadge.vue'
import { useBreadcrumbs } from '@/composables/useBreadcrumbs'
import { localizeError } from '@/i18n'

const props = defineProps<{ slug: string }>()
const router = useRouter()
const { setBreadcrumbs, clearBreadcrumbs } = useBreadcrumbs()

const org = ref<Organization | null>(null)
const repos = ref<Repository[]>([])
const members = ref<OrgMember[]>([])
const tab = ref<'repos' | 'members' | 'settings'>('repos')
const loading = ref(true)
const error = ref('')
const showRepoModal = ref(false)
const memberUsername = ref('')
const memberRole = ref('member')
const busy = ref(false)
const name = ref('')
const description = ref('')

const canAdmin = computed(() => org.value?.role === 'owner' || org.value?.role === 'admin')
const canOwner = computed(() => org.value?.role === 'owner')

function repoTo(repo: Repository) {
  const path = repo.folder_path ? `${repo.folder_path}/${repo.name}` : repo.name
  return { 
    name: 'repo' as const,
    params: { 
      owner: repo.owner, 
      repoPath: path 
    } 
  }
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    org.value = await orgsApi.get(props.slug)
    name.value = org.value.name
    description.value = org.value.description
    setBreadcrumbs([
      { 
        label: 'orgs', 
        labelKey: 'nav.orgs', 
        to: { 
          name: 'orgs' 
        } 
      },
      { 
        label: org.value.slug 
      },
    ])
    const [repoRes, memberRes] = await Promise.all([
      orgsApi.listRepos(props.slug),
      org.value.role ? orgsApi.listMembers(props.slug) : Promise.resolve({ 
        members: [] as OrgMember[] 
      }),
    ])
    repos.value = repoRes.repos ?? []
    members.value = memberRes.members ?? []
  } catch (err) {
    error.value = localizeError(err, 'errors.loadOrg')
  } finally {
    loading.value = false
  }
}

async function addMember() {
  busy.value = true
  error.value = ''
  try {
    await orgsApi.addMember(props.slug, {
      username: memberUsername.value.trim(),
      role: memberRole.value,
    })
    memberUsername.value = ''
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.orgMember')
  } finally {
    busy.value = false
  }
}

async function saveSettings() {
  busy.value = true
  error.value = ''
  try {
    org.value = await orgsApi.update(props.slug, {
      name: name.value.trim(),
      description: description.value.trim(),
    })
  } catch (err) {
    error.value = localizeError(err, 'errors.saveOrg')
  } finally {
    busy.value = false
  }
}

async function removeOrg() {
  if (!confirm(`Delete organization ${props.slug}?`)) {
    return
  }
  busy.value = true
  try {
    await orgsApi.remove(props.slug)
    await router.push({ name: 'orgs' })
  } catch (err) {
    error.value = localizeError(err, 'errors.deleteOrg')
  } finally {
    busy.value = false
  }
}

watch(() => props.slug, load)
onMounted(load)
onUnmounted(clearBreadcrumbs)
</script>

<template>
  <div>
    <PageState :loading="loading" :error="error">
      <template v-if="org">
        <div class="mb-6">
          <h1 class="text-2xl font-semibold">{{ org.name }}</h1>
          <p class="mt-1 font-mono text-sm text-ink-muted">/{{ org.slug }}</p>
          <p v-if="org.description" class="mt-2 text-sm text-ink-muted">{{ org.description }}</p>
        </div>

        <div class="mb-4 flex flex-wrap gap-2 border-b border-line pb-2">
          <button 
            type="button" 
            class="btn-ghost" 
            :class="{ 'bg-paper-2': tab === 'repos' }" 
            @click="tab = 'repos'"
          >
            {{ $t('orgs.repos') }}
          </button>
          <button
            v-if="org.role"
            type="button"
            class="btn-ghost"
            :class="{ 'bg-paper-2': tab === 'members' }"
            @click="tab = 'members'"
          >{{ $t('orgs.members') }}</button>
          <button
            v-if="canAdmin"
            type="button"
            class="btn-ghost"
            :class="{ 'bg-paper-2': tab === 'settings' }"
            @click="tab = 'settings'"
          >{{ $t('orgs.settings') }}</button>
        </div>

        <div v-if="tab === 'repos'">
          <div class="mb-3 flex justify-end">
            <button 
              v-if="canAdmin" 
              type="button" 
              class="btn-primary" 
              @click="showRepoModal = true"
            >
              {{ $t('home.newRepository') }}
            </button>
          </div>
          <p v-if="!repos.length" class="text-sm text-ink-muted">{{ $t('orgs.noRepos') }}</p>
          <ul v-else class="divide-y divide-line rounded-md border border-line bg-white">
            <li v-for="repo in repos" :key="repo.id">
              <RouterLink 
                :to="repoTo(repo)" 
                class="flex items-center justify-between gap-3 px-4 py-3 hover:bg-paper"
              >
                <span class="font-semibold">{{ repo.name }}</span>
                <VisibilityBadge :is-private="repo.is_private" />
              </RouterLink>
            </li>
          </ul>
        </div>

        <div v-else-if="tab === 'members'" class="space-y-4">
          <form 
            v-if="canAdmin" 
            class="flex flex-wrap items-end gap-2" 
            @submit.prevent="addMember"
          >
            <label class="block text-sm">
              <span class="mb-1 block font-semibold">{{ $t('orgs.username') }}</span>
              <input v-model="memberUsername" class="input" required>
            </label>
            <label class="block text-sm">
              <span class="mb-1 block font-semibold">{{ $t('orgs.role') }}</span>
              <select v-model="memberRole" class="input">
                <option value="member">member</option>
                <option value="admin">admin</option>
                <option v-if="canOwner" value="owner">owner</option>
              </select>
            </label>
            <button 
              type="submit" 
              class="btn-primary" 
              :disabled="busy"
            >{{ $t('orgs.addMember') }}</button>
          </form>
          <ul class="divide-y divide-line rounded-md border border-line bg-white">
            <li 
              v-for="m in members" 
              :key="m.user_id" 
              class="flex items-center justify-between px-4 py-3 text-sm"
            >
              <span class="font-semibold">{{ m.username }}</span>
              <span class="text-ink-muted">{{ m.role }}</span>
            </li>
          </ul>
        </div>

        <div 
          v-else-if="tab === 'settings'" 
          class="max-w-lg space-y-4"
        >
          <form class="space-y-3" @submit.prevent="saveSettings">
            <label class="block text-sm">
              <span class="mb-1 block font-semibold">{{ $t('orgs.name') }}</span>
              <input v-model="name" class="input" required>
            </label>
            <label class="block text-sm">
              <span class="mb-1 block font-semibold">{{ $t('orgs.description') }}</span>
              <input v-model="description" class="input">
            </label>
            <button 
              type="submit" 
              class="btn-primary" 
              :disabled="busy"
            >{{ $t('orgs.save') }}</button>
          </form>
          <div v-if="canOwner" class="rounded-md border border-warn/40 p-4">
            <p class="mb-2 text-sm text-warn">{{ $t('orgs.danger') }}</p>
            <button 
              type="button" 
              class="btn-danger" 
              :disabled="busy" 
              @click="removeOrg"
            >{{ $t('orgs.delete') }}</button>
          </div>
        </div>
      </template>
    </PageState>

    <CreateRepoModal
      v-if="showRepoModal"
      :organization="slug"
      @close="showRepoModal = false"
      @created="load"
    />
  </div>
</template>
