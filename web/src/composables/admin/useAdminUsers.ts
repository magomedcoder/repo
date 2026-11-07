import { onMounted, ref, watch } from 'vue'
import { adminUsersApi } from '@/api'
import { RoleAdmin, RoleUser, type AdminUser, type UserRole } from '@/api/types'
import { useAuth } from '@/composables/useAuth'
import { localizeError } from '@/i18n'

export function useAdminUsers() {
  const auth = useAuth()
  const users = ref<AdminUser[]>([])
  const total = ref(0)
  const q = ref('')
  const roleFilter = ref<'' | 'admin' | 'user'>('')
  const sort = ref('id')
  const error = ref('')
  const loading = ref(true)
  const saving = ref(false)
  const showForm = ref(false)
  const editing = ref<AdminUser | null>(null)
  const pendingDelete = ref<AdminUser | null>(null)
  const pendingStepUp = ref<null | {
    mode: 'create' | 'update'
    body: Record<string, unknown>
  }>(null)

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
    showForm.value = true
  }

  function openEdit(user: AdminUser) {
    editing.value = user
    formUsername.value = user.username
    formEmail.value = user.email
    formPassword.value = ''
    formRole.value = user.role
    showForm.value = true
  }

  function closeForm() {
    showForm.value = false
    resetForm()
  }

  async function load(reset = true) {
    loading.value = true
    error.value = ''
    try {
      const offset = reset ? 0 : users.value.length
      const res = await adminUsersApi.list({
        q: q.value.trim() || undefined,
        role: roleFilter.value || undefined,
        sort: sort.value || undefined,
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

        const emailChanged =
          formEmail.value.trim().toLowerCase() !== editing.value.email.trim().toLowerCase()
        const needsStepUp =
          !!formPassword.value || formRole.value !== editing.value.role || emailChanged
        if (needsStepUp) {
          pendingStepUp.value = { mode: 'update', body }
          saving.value = false
          return
        }

        await adminUsersApi.update(editing.value.id, body)
        if (auth.user.value?.id === editing.value.id) {
          await auth.fetchMe()
        }
      } else {
        pendingStepUp.value = {
          mode: 'create',
          body: {
            username: formUsername.value.trim(),
            email: formEmail.value.trim(),
            password: formPassword.value,
            role: formRole.value,
          },
        }
        saving.value = false
        return
      }
      closeForm()
      await load(true)
    } catch (err) {
      error.value = localizeError(err, editing.value ? 'errors.saveUser' : 'errors.createUser')
    } finally {
      saving.value = false
    }
  }

  async function confirmStepUp(password: string) {
    if (!pendingStepUp.value) {
      return
    }

    saving.value = true
    error.value = ''
    try {
      if (pendingStepUp.value.mode === 'create') {
        await adminUsersApi.create({
          username: String(pendingStepUp.value.body.username ?? ''),
          email: String(pendingStepUp.value.body.email ?? ''),
          password: String(pendingStepUp.value.body.password ?? ''),
          role: pendingStepUp.value.body.role as UserRole,
          current_password: password,
        })
      } else {
        if (!editing.value) {
          return
        }
        await adminUsersApi.update(editing.value.id, {
          ...pendingStepUp.value.body,
          current_password: password,
        })
        if (auth.user.value?.id === editing.value.id) {
          await auth.fetchMe()
        }
      }
      pendingStepUp.value = null
      closeForm()
      await load(true)
    } catch (err) {
      error.value = localizeError(err, pendingStepUp.value?.mode === 'create' ? 'errors.createUser' : 'errors.saveUser')
    } finally {
      saving.value = false
    }
  }

  function askDelete(user: AdminUser) {
    pendingDelete.value = user
  }

  async function confirmDelete(password: string) {
    if (!pendingDelete.value) {
      return
    }
    
    error.value = ''
    try {
      await adminUsersApi.remove(pendingDelete.value.id, password)
      pendingDelete.value = null
      await load(true)
    } catch (err) {
      error.value = localizeError(err, 'errors.deleteFailed')
    }
  }

  let searchTimer: ReturnType<typeof setTimeout> | undefined
  watch([q, roleFilter, sort], () => {
    clearTimeout(searchTimer)
    searchTimer = setTimeout(() => {
      void load(true)
    }, 250)
  })

  onMounted(() => load(true))

  return {
    RoleAdmin,
    RoleUser,
    users,
    total,
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
    load,
    submit,
    confirmStepUp,
    askDelete,
    confirmDelete,
  }
}
