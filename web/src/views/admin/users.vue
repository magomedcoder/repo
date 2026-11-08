<script setup lang="ts">
import CpConfirmPassword from '@/components/admin/CpConfirmPassword.vue'
import CpPage from '@/components/admin/CpPage.vue'
import CpPagination from '@/components/admin/CpPagination.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import Icon from '@/components/ui/Icon.vue'
import ModalForm from '@/components/ui/ModalForm.vue'
import PageState from '@/components/ui/PageState.vue'
import Surface from '@/components/ui/Surface.vue'
import { useAdminUsers } from '@/composables/admin/useAdminUsers'
import { formatDate } from '@/i18n'

const {
  RoleAdmin,
  RoleUser,
  users,
  total,
  page,
  pageSize,
  q,
  roleFilter,
  sort,
  error,
  loading,
  saving,
  showForm,
  editing,
  pendingDelete,
  pendingStepUp,
  formUsername,
  formEmail,
  formPassword,
  formRole,
  openCreate,
  openEdit,
  closeForm,
  setPage,
  submit,
  confirmStepUp,
  askDelete,
  confirmDelete,
} = useAdminUsers()
</script>

<template>
  <CpPage :title="$t('cp.users.title')" :description="$t('cp.users.description')">
    <template #actions>
      <button type="button" class="btn-primary" @click="openCreate">
        <Icon name="plus" size="sm" />
        {{ $t('users.add') }}
      </button>
    </template>

    <div class="mb-3 flex flex-wrap gap-2">
      <input v-model="q" type="search" class="field min-w-[12rem] flex-1" :placeholder="$t('users.search')">
      <select v-model="roleFilter" class="field w-auto">
        <option value="">{{ $t('cp.users.allRoles') }}</option>
        <option value="admin">{{ $t('users.roleAdmin') }}</option>
        <option value="user">{{ $t('users.roleUser') }}</option>
      </select>
      <select v-model="sort" class="field w-auto">
        <option value="id">{{ $t('cp.users.sortId') }}</option>
        <option value="username">{{ $t('cp.users.sortName') }}</option>
        <option value="created">{{ $t('cp.users.sortCreated') }}</option>
        <option value="role">{{ $t('cp.users.sortRole') }}</option>
      </select>
    </div>

    <p v-if="error" class="mb-3 text-sm text-warn">{{ error }}</p>

    <PageState :loading="loading && !users.length">
      <Surface class="overflow-x-auto">
        <table class="w-full min-w-[36rem] border-collapse text-left text-[12px]">
          <thead class="sticky top-0 bg-white/55 text-ink backdrop-blur-sm">
            <tr>
              <th class="border-b border-line px-3 py-2 font-semibold">{{ $t('users.username') }}</th>
              <th class="border-b border-line px-3 py-2 font-semibold">{{ $t('users.email') }}</th>
              <th class="border-b border-line px-3 py-2 font-semibold">{{ $t('users.role') }}</th>
              <th class="border-b border-line px-3 py-2 font-semibold">{{ $t('users.created') }}</th>
              <th class="border-b border-line px-3 py-2" />
            </tr>
          </thead>
          <tbody>
            <tr v-for="user in users" :key="user.id" class="odd:bg-transparent even:bg-row-alt">
              <td class="border-b border-line px-3 py-2 font-semibold text-ink">{{ user.username }}</td>
              <td class="border-b border-line px-3 py-2 text-ink-muted">{{ user.email }}</td>
              <td class="border-b border-line px-3 py-2">
                <span
                  class="inline-flex rounded-full border px-1.5 py-px text-[10px] font-medium"
                  :class="user.role === RoleAdmin ? 'border-ink/30 bg-white/60 text-ink' : 'border-line-strong/70 text-ink-muted'"
                >
                  {{ user.role === RoleAdmin ? $t('users.roleAdmin') : $t('users.roleUser') }}
                </span>
              </td>
              <td class="border-b border-line px-3 py-2 text-ink-muted">{{ formatDate(user.created_at) }}</td>
              <td class="border-b border-line px-3 py-2">
                <div class="flex flex-wrap justify-end gap-1.5">
                  <button type="button" class="btn" @click="openEdit(user)">
                    <Icon name="pencil" size="sm" />
                    {{ $t('users.edit') }}
                  </button>
                  <button type="button" class="btn-warn" @click="askDelete(user)">
                    <Icon name="trash" size="sm" />
                    {{ $t('users.delete') }}
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
        <EmptyState v-if="!users.length" :message="$t('users.empty')" />
      </Surface>
      <CpPagination
        :page="page"
        :page-size="pageSize"
        :total="total"
        :loading="loading"
        @update:page="setPage"
      />
    </PageState>

    <ModalForm
      v-if="showForm"
      :title="editing ? $t('users.edit') : $t('users.add')"
      :busy="saving"
      :submit-label="saving
        ? (editing ? $t('users.saving') : $t('users.creating'))
        : (editing ? $t('users.save') : $t('users.create'))"
      @close="closeForm"
      @submit="submit"
    >
      <div class="grid gap-3 sm:grid-cols-2">
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('users.username') }}</span>
          <input v-model="formUsername" class="field" required :disabled="!!editing">
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ $t('users.email') }}</span>
          <input v-model="formEmail" type="email" class="field" required>
        </label>
        <label class="block text-sm">
          <span class="mb-1 block font-semibold">{{ editing ? $t('users.newPassword') : $t('users.password') }}</span>
          <input
            v-model="formPassword"
            type="password"
            class="field"
            :required="!editing"
            minlength="10"
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
      </div>
    </ModalForm>

    <CpConfirmPassword
      v-if="pendingStepUp"
      :title="$t('cp.users.confirmStepUp')"
      :message="$t('cp.users.confirmStepUpHint')"
      @close="pendingStepUp = null"
      @confirm="confirmStepUp"
    />

    <CpConfirmPassword
      v-if="pendingDelete"
      :title="$t('users.delete')"
      :message="$t('users.confirmDelete', { username: pendingDelete.username })"
      @close="pendingDelete = null"
      @confirm="confirmDelete"
    />
  </CpPage>
</template>
