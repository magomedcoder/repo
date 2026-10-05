<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { orgsApi } from '@/api'
import type { Organization } from '@/api/types'
import PageState from '@/components/ui/PageState.vue'
import { useBreadcrumbs } from '@/composables/useBreadcrumbs'
import { localizeError } from '@/i18n'

const { setBreadcrumbs, clearBreadcrumbs } = useBreadcrumbs()

const orgs = ref<Organization[]>([])
const loading = ref(true)
const error = ref('')
const showForm = ref(false)
const slug = ref('')
const name = ref('')
const description = ref('')
const creating = ref(false)

async function load() {
  loading.value = true
  error.value = ''
  try {
    setBreadcrumbs([{ 
      label: 'orgs', 
      labelKey: 'nav.orgs' 
    }])
    const res = await orgsApi.listMine()
    orgs.value = res.organizations ?? []
  } catch (err) {
    error.value = localizeError(err, 'errors.loadOrgs')
  } finally {
    loading.value = false
  }
}

async function create() {
  creating.value = true
  error.value = ''
  try {
    await orgsApi.create({
      slug: slug.value.trim(),
      name: name.value.trim(),
      description: description.value.trim(),
    })
    slug.value = ''
    name.value = ''
    description.value = ''
    showForm.value = false
    await load()
  } catch (err) {
    error.value = localizeError(err, 'errors.createOrg')
  } finally {
    creating.value = false
  }
}

onMounted(load)
onUnmounted(clearBreadcrumbs)
</script>

<template>
  <div>
    <div class="mb-6 flex flex-wrap items-end justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold">{{ $t('orgs.title') }}</h1>
        <p class="mt-1 text-sm text-ink-muted">{{ $t('orgs.lead') }}</p>
      </div>
      <button 
        type="button" 
        class="btn-primary" 
        @click="showForm = !showForm"
      >
        {{ $t('orgs.new') }}
      </button>
    </div>

    <form
      v-if="showForm"
      class="mb-6 space-y-3 rounded-md border border-line bg-white p-4"
      @submit.prevent="create"
    >
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('orgs.slug') }}</span>
        <input 
          v-model="slug" 
          class="input font-mono" 
          required 
          minlength="3" 
          maxlength="39" 
          pattern="[a-z0-9_-]+" 
          placeholder="name"
        >
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('orgs.name') }}</span>
        <input 
          v-model="name" 
          class="input" 
          required 
          maxlength="100" 
          placeholder="Name"
        >
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('orgs.description') }}</span>
        <input 
          v-model="description" 
          class="input" 
          :placeholder="$t('modal.optional')"
        >
      </label>
      <button 
        type="submit" 
        class="btn-primary" 
        :disabled="creating"
      >
        {{ creating ? $t('orgs.creating') : $t('orgs.create') }}
      </button>
    </form>

    <PageState :loading="loading" :error="error">
      <p v-if="!orgs.length" class="text-sm text-ink-muted">{{ $t('orgs.empty') }}</p>
      <ul v-else class="divide-y divide-line rounded-md border border-line bg-white">
        <li v-for="org in orgs" :key="org.id">
          <RouterLink
            :to="{ 
              name: 'org', 
              params: { 
                slug: org.slug 
              }
            }"
            class="flex items-center justify-between gap-3 px-4 py-3 hover:bg-paper"
          >
            <div class="min-w-0">
              <p class="truncate font-semibold">{{ org.name }}</p>
              <p class="truncate text-sm text-ink-muted">/{{ org.slug }}</p>
            </div>
            <span v-if="org.role" class="text-xs text-ink-muted">{{ org.role }}</span>
          </RouterLink>
        </li>
      </ul>
    </PageState>
  </div>
</template>
