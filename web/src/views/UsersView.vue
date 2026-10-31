<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { adminUsersApi } from '@/api'
import type { AdminUser, UserRole } from '@/api/types'
import PageState from '@/components/ui/PageState.vue'
import { useAuth } from '@/composables/useAuth'
import { formatDate, i18n, localizeError } from '@/i18n'

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
const formRole = ref<UserRole>('user')

const pageSize = 50

function resetForm() {
  editing.value = null
  formUsername.value = ''
  formEmail.value = ''
  formPassword.value = ''
  formRole.value = 'user'
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
      const body: { email?: string; role?: UserRole; password?: string } = {
        email: formEmail.value.trim(),
        role: formRole.value,
      }
      
      if (formPassword.value) {
        body.password = formPassword.value
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

  error.value = ''
  try {
    await adminUsersApi.remove(user.id)
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
    <div class="mb-4 flex flex-wrap items-start justify-between gap-3 border-b border-line pb-3">
      <div>
        <h1 class="text-[140%] font-bold text-ink">{{ $t('users.title') }}</h1>
      </div>
      <button
        type="button"
        class="inline-flex items-center justify-center border border-ink bg-ink px-2.5 py-1 text-[10pt] text-paper hover:bg-ink-muted disabled:cursor-not-allowed disabled:opacity-50"
        @click="openCreate"
      >
        {{ $t('users.add') }}
      </button>
    </div>

    <label class="mb-3 block text-sm">
      <span class="sr-only">{{ $t('users.search') }}</span>
      <input
        v-model="q"
        type="search"
        class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none"
        :placeholder="$t('users.search')"
      >
    </label>

    <form
      v-if="showForm"
      class="overflow-hidden border border-line bg-paper mb-4"
      @submit.prevent="submit"
    >
      <div class="flex items-center gap-2 border-b border-line bg-path px-3 py-1.5 text-[10pt] text-ink">
        {{ editing ? $t('users.edit') : $t('users.add') }}
      </div>
      <div class="p-3 sm:p-4 space-y-3">
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('users.username') }}</span>
          <input
            v-model="formUsername"
            class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none"
            required
            :disabled="!!editing"
          >
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('users.email') }}</span>
          <input
            v-model="formEmail"
            type="email"
            class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none"
            required
          >
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ editing ? $t('users.newPassword') : $t('users.password') }}</span>
          <input
            v-model="formPassword"
            type="password"
            class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none"
            :required="!editing"
            minlength="8"
            autocomplete="new-password"
          >
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('users.role') }}</span>
          <select v-model="formRole" class="w-full border border-line bg-paper px-2 py-1 text-[10pt] text-ink outline-none">
            <option value="user">{{ $t('users.roleUser') }}</option>
            <option value="admin">{{ $t('users.roleAdmin') }}</option>
          </select>
        </label>
        <button
          type="submit"
          class="inline-flex items-center justify-center border border-ink bg-ink px-2.5 py-1 text-[10pt] text-paper hover:bg-ink-muted disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="saving"
        >
          {{ saving ? (editing ? $t('users.saving') : $t('users.creating')) : (editing ? $t('users.save') : $t('users.create')) }}
        </button>
      </div>
    </form>

    <p v-if="error" class="mb-3 text-sm text-warn">{{ error }}</p>
    <PageState :loading="loading && !users.length">
      <ul class="overflow-hidden border border-line bg-paper divide-y divide-line">
        <li
          v-for="user in users"
          :key="user.id"
          class="flex flex-wrap items-start justify-between gap-3 px-4 py-3"
        >
          <div class="min-w-0">
            <p class="font-semibold text-ink">
              {{ user.username }}
              <span class="ml-2 font-normal text-ink-muted">{{ user.role === 'admin' ? $t('users.roleAdmin') : $t('users.roleUser') }}</span>
            </p>
            <p class="mt-0.5 text-xs text-ink-muted">
              {{ user.email }} {{ $t('users.created') }} {{ formatDate(user.created_at) }}
            </p>
          </div>
          <div class="flex flex-wrap gap-2">
            <button
              type="button"
              class="inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-ink hover:bg-row-hover disabled:cursor-not-allowed disabled:opacity-50"
              @click="openEdit(user)"
            >
              {{ $t('users.edit') }}
            </button>
            <button
              type="button"
              class="inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-warn hover:bg-warn-soft disabled:cursor-not-allowed disabled:opacity-50"
              @click="remove(user)"
            >
              {{ $t('users.delete') }}
            </button>
          </div>
        </li>
        <li v-if="!users.length" class="px-4 py-10 text-center text-sm text-ink-muted">{{ $t('users.empty') }}</li>
      </ul>
      <p v-if="users.length < total" class="mt-3">
        <button
          type="button"
          class="inline-flex items-center justify-center border border-line bg-paper px-2.5 py-1 text-[10pt] text-ink hover:bg-row-hover disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="loading"
          @click="load(false)"
        >
          {{ loading ? $t('common.loading') : $t('users.loadMore') }}
        </button>
      </p>
    </PageState>
  </div>
</template>
