<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { adminUsersApi } from '@/api'
import { RoleAdmin, RoleUser, type AdminUser, type UserRole } from '@/api/types'
import PageState from '@/components/ui/PageState.vue'
import { useAuth } from '@/composables/useAuth'
import { formatDate, i18n, localizeError } from '@/i18n'
import Icon from '@/components/ui/Icon.vue'
import PageHeader from '@/components/ui/PageHeader.vue'
import Surface from '@/components/ui/Surface.vue'
import SurfaceHeader from '@/components/ui/SurfaceHeader.vue'
import EmptyState from '@/components/ui/EmptyState.vue'

const auth = useAuth()
const users = ref<AdminUser[]>([])
const total = ref(0)
const q = ref('')
const error = ref('')
const loading = ref(true)
const saving = ref(false)
const showForm = ref(false)
const editing = ref<AdminUser | null>(null)

const formUsername = ref('')
const formEmail = ref('')
const formPassword = ref('')
const formRole = ref<UserRole>(RoleUser)

const pageSize = 50

function resetForm() {
  editing.value = null
  formUsername.value = ''
  formEmail.value = ''
  formPassword.value = ''
  formRole.value = RoleUser
}

function openCreate() {
  resetForm()
  showForm.value = !showForm.value
  if (!showForm.value) {
    return
  }
}

function openEdit(user: AdminUser) {
  editing.value = user
  formUsername.value = user.username
  formEmail.value = user.email
  formPassword.value = ''
  formRole.value = user.role
  showForm.value = true
}

async function load(reset = true) {
  loading.value = true
  error.value = ''
  try {
    const offset = reset ? 0 : users.value.length
    const res = await adminUsersApi.list({
      q: q.value.trim() || undefined,
      limit: pageSize,
      offset,
    })
    total.value = res.total
    users.value = reset ? (res.users ?? []) : [...users.value, ...(res.users ?? [])]
  } catch (err) {
    error.value = localizeError(err, 'errors.loadUsers')
  } finally {
    loading.value = false
  }
}

async function submit() {
  saving.value = true
  error.value = ''
  try {
    if (editing.value) {
      const body: { 
        email?: string
        role?: UserRole
        password?: string
        current_password?: string
      } = {
        email: formEmail.value.trim(),
        role: formRole.value,
      }

      if (formPassword.value) {
        body.password = formPassword.value
      }

      const needsStepUp = !!formPassword.value || formRole.value !== editing.value.role
      if (needsStepUp) {
        const current = window.prompt(i18n.global.t('users.currentPassword'), '')
        if (!current) {
          saving.value = false
          return
        }
        
        body.current_password = current
      }

      await adminUsersApi.update(editing.value.id, body)

      if (auth.user.value?.id === editing.value.id) {
        await auth.fetchMe()
      }
    } else {
      await adminUsersApi.create({
        username: formUsername.value.trim(),
        email: formEmail.value.trim(),
        password: formPassword.value,
        role: formRole.value,
      })
    }
    showForm.value = false
    resetForm()
    await load(true)
  } catch (err) {
    error.value = localizeError(err, editing.value ? 'errors.saveUser' : 'errors.createUser')
  } finally {
    saving.value = false
  }
}

async function remove(user: AdminUser) {
  if (!confirm(i18n.global.t('users.confirmDelete', { username: user.username }))) {
    return
  }

  const current = window.prompt(i18n.global.t('users.currentPassword'), '')
  if (!current) {
    return
  }

  error.value = ''
  try {
    await adminUsersApi.remove(user.id, current)
    await load(true)
  } catch (err) {
    error.value = localizeError(err, 'errors.deleteFailed')
  }
}

let searchTimer: ReturnType<typeof setTimeout> | undefined
watch(q, () => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    void load(true)
  }, 250)
})

onMounted(() => load(true))
</script>

<template>
  <div class="mx-auto max-w-3xl">
    <PageHeader :title="$t('users.title')" icon="users">
      <template #actions>
        <button
          type="button"
          class="btn-primary"
          @click="openCreate"
        >
          <Icon name="plus" size="sm" />
          {{ $t('users.add') }}
        </button>
      </template>
    </PageHeader>

    <label class="mb-3 block text-sm">
      <span class="sr-only">{{ $t('users.search') }}</span>
      <input
        v-model="q"
        type="search"
        class="field"
        :placeholder="$t('users.search')"
      >
    </label>

    <Surface
      v-if="showForm"
      as="form"
      class="mb-4"
      @submit.prevent="submit"
    >
      <SurfaceHeader>
        {{ editing ? $t('users.edit') : $t('users.add') }}
      </SurfaceHeader>
      <div class="p-3 sm:p-4 space-y-3">
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('users.username') }}</span>
          <input
            v-model="formUsername"
            class="field"
            required
            :disabled="!!editing"
          >
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('users.email') }}</span>
          <input
            v-model="formEmail"
            type="email"
            class="field"
            required
          >
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ editing ? $t('users.newPassword') : $t('users.password') }}</span>
          <input
            v-model="formPassword"
            type="password"
            class="field"
            :required="!editing"
            minlength="8"
            autocomplete="new-password"
          >
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('users.role') }}</span>
          <select v-model.number="formRole" class="field">
            <option :value="RoleUser">{{ $t('users.roleUser') }}</option>
            <option :value="RoleAdmin">{{ $t('users.roleAdmin') }}</option>
          </select>
        </label>
        <button
          type="submit"
          class="btn-primary"
          :disabled="saving"
        >
          {{ saving ? (editing ? $t('users.saving') : $t('users.creating')) : (editing ? $t('users.save') : $t('users.create')) }}
        </button>
      </div>
    </Surface>

    <p v-if="error" class="mb-3 text-sm text-warn">{{ error }}</p>
    <PageState :loading="loading && !users.length">
      <Surface as="ul" class="divide-y divide-line">
        <li
          v-for="user in users"
          :key="user.id"
          class="flex flex-wrap items-start justify-between gap-3 px-4 py-3"
        >
          <div class="min-w-0">
            <p class="font-semibold text-ink">
              {{ user.username }}
              <span class="ml-2 font-normal text-ink-muted">{{ user.role === RoleAdmin ? $t('users.roleAdmin') : $t('users.roleUser') }}</span>
            </p>
            <p class="mt-0.5 text-xs text-ink-muted">
              {{ user.email }} {{ $t('users.created') }} {{ formatDate(user.created_at) }}
            </p>
          </div>
          <div class="flex flex-wrap gap-2">
            <button
              type="button"
              class="btn"
              @click="openEdit(user)"
            >
              <Icon name="pencil" size="sm" />
              {{ $t('users.edit') }}
            </button>
            <button
              type="button"
              class="btn-warn"
              @click="remove(user)"
            >
              <Icon name="trash" size="sm" />
              {{ $t('users.delete') }}
            </button>
          </div>
        </li>
        <EmptyState
          v-if="!users.length"
          as="li"
          :message="$t('users.empty')"
        />
      </Surface>
      <p v-if="users.length < total" class="mt-3">
        <button
          type="button"
          class="btn"
          :disabled="loading"
          @click="load(false)"
        >
          {{ loading ? $t('common.loading') : $t('users.loadMore') }}
        </button>
      </p>
    </PageState>
  </div>
</template>
