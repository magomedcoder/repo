import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { accessApi, reposApi } from '@/api'
import { repoLocation } from '@/lib/repoNav'
import { useAuth } from '@/composables/useAuth'
import { useRegistration } from '@/composables/useRegistration'
import { useSite } from '@/composables/useSite'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/auth/login.vue'),
    meta: { guest: true },
  },
  {
    path: '/register',
    name: 'register',
    component: () => import('@/views/auth/register.vue'),
    meta: { guest: true },
  },
  {
    path: '/',
    component: () => import('@/layouts/AppLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        name: 'home',
        meta: { publicRead: true },
        component: () => import('@/views/home.vue'),
      },
      {
        path: 'profile',
        name: 'profile',
        component: () => import('@/views/profile/index.vue'),
      },
      {
        path: 'profile/email',
        name: 'profile-email',
        component: () => import('@/views/profile/email.vue'),
      },
      {
        path: 'profile/password',
        name: 'profile-password',
        component: () => import('@/views/profile/password.vue'),
      },
      {
        path: 'profile/sessions',
        name: 'sessions',
        component: () => import('@/views/profile/sessions.vue'),
      },
      {
        path: 'profile/tokens',
        name: 'tokens',
        component: () => import('@/views/profile/tokens.vue'),
      },
      {
        path: 'settings/teams',
        name: 'teams',
        component: () => import('@/views/profile/teams.vue'),
      },
      {
        path: 'settings/teams/:slug',
        name: 'team',
        component: () => import('@/views/profile/team.vue'),
        props: true,
      },
      {
        path: 'cp',
        component: () => import('@/layouts/CpLayout.vue'),
        meta: { requiresAdmin: true },
        children: [
          {
            path: '',
            name: 'cp',
            component: () => import('@/views/admin/overview.vue'),
          },
          {
            path: 'users',
            name: 'users',
            component: () => import('@/views/admin/users.vue'),
          },
          {
            path: 'access-logs',
            name: 'access-logs',
            component: () => import('@/views/admin/access-logs.vue'),
          },
          {
            path: 'mirrors',
            name: 'admin-mirrors',
            component: () => import('@/views/admin/mirrors.vue'),
          },
          {
            path: 'repos',
            name: 'cp-repos',
            component: () => import('@/views/admin/repos.vue'),
          },
        ],
      },
      {
        path: 'search',
        name: 'search',
        meta: { publicRead: true },
        component: () => import('@/views/search.vue'),
      },
      {
        path: ':repoPath/pulls/:number',
        name: 'repo-pull',
        meta: { publicRead: true },
        component: () => import('@/views/repo/pull.vue'),
        props: true,
      },
      {
        path: ':repoPath/pulls',
        name: 'repo-pulls',
        meta: { publicRead: true },
        component: () => import('@/views/repo/pulls.vue'),
        props: true,
      },
      {
        path: ':repoPath/issues/:number',
        name: 'repo-issue',
        meta: { publicRead: true },
        component: () => import('@/views/repo/issue.vue'),
        props: true,
      },
      {
        path: ':repoPath/issues',
        name: 'repo-issues',
        meta: { publicRead: true },
        component: () => import('@/views/repo/issues.vue'),
        props: true,
      },
      {
        path: ':repoPath/commits/:sha',
        name: 'repo-commit',
        meta: { publicRead: true },
        component: () => import('@/views/repo/commit.vue'),
        props: true,
      },
      {
        path: ':repoPath/refs',
        name: 'repo-refs',
        meta: { publicRead: true },
        component: () => import('@/views/repo/refs.vue'),
        props: true,
      },
      {
        path: ':repoPath/commits',
        name: 'repo-commits',
        meta: { publicRead: true },
        component: () => import('@/views/repo/commits.vue'),
        props: true,
      },
      {
        path: ':repoPath/blob',
        name: 'repo-blob',
        meta: { publicRead: true },
        component: () => import('@/views/repo/blob.vue'),
        props: true,
      },
      {
        path: ':repoPath/settings',
        name: 'repo-settings',
        meta: { requiresRepoAdmin: true },
        component: () => import('@/views/repo/settings.vue'),
        props: true,
      },
      {
        path: ':repoPath',
        name: 'repo',
        meta: { publicRead: true },
        component: () => import('@/views/repo/home.vue'),
        props: true,
      },
      {
        path: ':pathMatch(.*)*',
        name: 'not-found',
        meta: { publicRead: true },
        component: () => import('@/views/not-found.vue'),
      },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior() {
    return { top: 0 }
  },
})

router.beforeEach(async (to) => {
  const site = useSite()
  if (!site.ready.value) {
    await site.fetchSite()
  }

  const auth = useAuth()
  if (!auth.ready.value) {
    await auth.fetchMe()
  }

  const publicRead = to.matched.some((record) => record.meta.publicRead)
  if (to.meta.requiresAuth && !auth.isAuthenticated.value && !publicRead) {
    return {
      name: 'login',
      query: {
        redirect: to.fullPath
      }
    }
  }

  const requiresAdmin = to.matched.some((record) => record.meta.requiresAdmin)
  if (requiresAdmin && !auth.isAdmin.value) {
    return { name: 'not-found' }
  }

  const requiresRepoAdmin = to.matched.some((record) => record.meta.requiresRepoAdmin)
  if (requiresRepoAdmin) {
    if (!auth.isAuthenticated.value) {
      return {
        name: 'login',
        query: {
          redirect: to.fullPath,
        },
      }
    }

    const repoPath = String(to.params.repoPath || '')
    if (repoPath) {
      try {
        const repo = await reposApi.get(repoPath)
        if (!repo.can_admin) {
          return repoLocation('repo', repoPath)
        }
      } catch {
        return { name: 'not-found' }
      }
    }
  }

  if (to.meta.guest && auth.isAuthenticated.value) {
    return { name: 'home' }
  }

  if (to.name === 'register') {
    const { fetchAllowRegistration } = useRegistration()
    const allowed = await fetchAllowRegistration()
    if (!allowed) {
      return { name: 'login' }
    }
  }

  return true
})

let skipNextVisitBeacon = true
router.afterEach((to) => {
  if (skipNextVisitBeacon) {
    skipNextVisitBeacon = false
    return
  }
  const auth = useAuth()
  if (!auth.isAuthenticated.value) {
    return
  }
  void accessApi.visit(to.path)
})

export default router
