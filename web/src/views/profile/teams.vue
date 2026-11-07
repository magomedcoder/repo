<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { teamsApi } from '@/api'
import type { Team } from '@/api/types'
import EmptyState from '@/components/ui/EmptyState.vue'
import Icon from '@/components/ui/Icon.vue'
import PageHeader from '@/components/ui/PageHeader.vue'
import PageState from '@/components/ui/PageState.vue'
import Surface from '@/components/ui/Surface.vue'
import SurfaceHeader from '@/components/ui/SurfaceHeader.vue'
import { formatDate, i18n, localizeError } from '@/i18n'

const teams = ref<Team[]>([])
const name = ref('')
const error = ref('')
const loading = ref(true)
const busy = ref(false)
const showForm = ref(false)

async function load() {
  loading.value = true
  error.value = ''

  try {
    const res = await teamsApi.list()
    teams.value = (res.teams ?? []).slice().sort((a, b) => a.name.localeCompare(b.name))
  } catch (err) {
    error.value = localizeError(err, 'errors.loadFailed')
  } finally {
    loading.value = false
  }
}

async function create() {
  busy.value = true
  error.value = ''

  try {
    await teamsApi.create({
      name: name.value.trim()
    })
    name.value = ''
    showForm.value = false
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.saveFailed')
  } finally {
    busy.value = false
  }
}

async function remove(team: Team) {
  if (!confirm(i18n.global.t('teams.confirmDelete', { name: team.name }))) {
    return
  }

  busy.value = true
  error.value = ''
  try {
    await teamsApi.remove(team.slug)
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.deleteFailed')
  } finally {
    busy.value = false
  }
}

onMounted(load)
</script>

<template>
  <div>
    <PageHeader :title="$t('teams.title')" icon="users" :hint="$t('teams.hint')">
      <template #actions>
        <button type="button" class="btn-primary" @click="showForm = !showForm">
          <Icon name="plus" size="sm" />
          {{ $t('teams.add') }}
        </button>
      </template>
    </PageHeader>

    <Surface
      v-if="showForm"
      as="form"
      class="mb-4"
      @submit.prevent="create"
    >
      <SurfaceHeader>{{ $t('teams.add') }}</SurfaceHeader>
      <div class="space-y-3 p-3 sm:p-4">
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('teams.name') }}</span>
          <input
            v-model="name"
            class="field"
            required
            maxlength="64"
            :placeholder="$t('teams.namePlaceholder')"
          >
        </label>
        <button type="submit" class="btn-primary disabled:opacity-50" :disabled="busy">
          {{ $t('teams.create') }}
        </button>
      </div>
    </Surface>

    <p v-if="error" class="mb-3 text-sm text-warn">{{ error }}</p>
    <PageState :loading="loading">
      <Surface as="ul" class="divide-y divide-line">
        <li
          v-for="team in teams"
          :key="team.slug"
          class="flex flex-wrap items-center justify-between gap-3 px-4 py-3"
        >
          <div class="min-w-0">
            <RouterLink
              :to="{ name: 'team', params: { slug: team.slug } }"
              class="inline-flex items-center gap-1.5 font-semibold"
            >
              <Icon name="users" size="sm" class="text-ink-muted" />
              {{ team.name }}
            </RouterLink>
            <p class="mt-0.5 font-mono text-xs text-ink-muted">{{ team.slug }}</p>
            <p class="mt-0.5 text-xs text-ink-muted">{{ formatDate(team.created_at) }}</p>
          </div>
          <button type="button" class="btn-warn" :disabled="busy" @click="remove(team)">
            <Icon name="trash" size="sm" />
            {{ $t('teams.delete') }}
          </button>
        </li>
        <EmptyState
          v-if="!teams.length"
          as="li"
          :message="$t('teams.empty')"
        />
      </Surface>
    </PageState>
  </div>
</template>
