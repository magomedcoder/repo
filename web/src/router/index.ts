import { createRouter, createWebHistory, type RouteLocationNormalized, type RouteRecordRaw } from 'vue-router'
import { accessApi, reposApi } from '@/api'
import { repoLocation } from '@/lib/repoNav'
import { useAuth } from '@/composables/useAuth'
import { useRegistration } from '@/composables/useRegistration'
import { useSite } from '@/composables/useSite'

const REPO_TAB_ROUTES: Record<string, string> = {
  settings: 'repo-settings',
  commits: 'repo-commits',
  blob: 'repo-blob',
  refs: 'repo-refs',
  issues: 'repo-issues',
  pulls: 'repo-pulls',
}

function redirectReservedRepoTab(to: RouteLocationNormalized) {
  const name = String(to.params.name || '')
  const routeName = REPO_TAB_ROUTES[name]
  if (!routeName) {
    return true
  }

  return {
    name: routeName,
    params: { repoPath: String(to.params.folder || '') },
    query: to.query,
    hash: to.hash,
  }
}

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/LoginView.vue'),
    meta: { guest: true },
  },
  {
    path: '/register',
    name: 'register',
    component: () => import('@/views/RegisterView.vue'),
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
        component: () => import('@/views/HomeView.vue'),
      },
      {
        path: 'folders',
        name: 'folders',
        component: () => import('@/views/FoldersView.vue'),
      },
      {
        path: 'settings/tokens',
        name: 'tokens',
        component: () => import('@/views/TokensView.vue'),
      },
      {
        path: 'settings/sessions',
        name: 'sessions',
        component: () => import('@/views/SessionsView.vue'),
      },
      {
        path: 'settings/profile',
        name: 'profile',
        component: () => import('@/views/ProfileView.vue'),
      },
      {
        path: 'settings/profile/email',
        name: 'profile-email',
        component: () => import('@/views/ProfileEmailView.vue'),
      },
      {
        path: 'settings/profile/password',
        name: 'profile-password',
        component: () => import('@/views/ProfilePasswordView.vue'),
      },
      {
        path: 'settings/users',
        name: 'users',
        meta: { requiresAdmin: true },
        component: () => import('@/views/UsersView.vue'),
      },
      {
        path: 'settings/access-logs',
        name: 'access-logs',
        meta: { requiresAdmin: true },
        component: () => import('@/views/AccessLogsView.vue'),
      },
      {
        path: 'settings/mirrors',
        name: 'admin-mirrors',
        meta: { requiresAdmin: true },
        component: () => import('@/views/AdminMirrorsView.vue'),
      },
      {
        path: 'search',
        name: 'search',
        meta: { publicRead: true },
        component: () => import('@/views/SearchView.vue'),
      },
      {
        path: ':repoPath(.*)/pulls/:number',
        name: 'repo-pull',
        meta: { publicRead: true },
        component: () => import('@/views/PullView.vue'),
        props: true,
      },
      {
        path: ':repoPath(.*)/pulls',
        name: 'repo-pulls',
        meta: { publicRead: true },
        component: () => import('@/views/PullsView.vue'),
        props: true,
      },
      {
        path: ':repoPath(.*)/issues/:number',
        name: 'repo-issue',
        meta: { publicRead: true },
        component: () => import('@/views/IssueView.vue'),
        props: true,
      },
      {
        path: ':repoPath(.*)/issues',
        name: 'repo-issues',
        meta: { publicRead: true },
        component: () => import('@/views/IssuesView.vue'),
        props: true,
      },
      {
        path: ':repoPath(.*)/commits/:sha',
        name: 'repo-commit',
        meta: { publicRead: true },
        component: () => import('@/views/RepoCommitView.vue'),
        props: true,
      },
      {
        path: ':repoPath(.*)/refs',
        name: 'repo-refs',
        meta: { publicRead: true },
        component: () => import('@/views/RepoRefsView.vue'),
        props: true,
      },
      {
        path: ':repoPath(.*)/commits',
        name: 'repo-commits',
        meta: { publicRead: true },
        component: () => import('@/views/RepoCommitsView.vue'),
        props: true,
      },
      {
        path: ':repoPath(.*)/blob',
        name: 'repo-blob',
        meta: { publicRead: true },
        component: () => import('@/views/RepoBlobView.vue'),
        props: true,
      },
      {
        path: ':repoPath(.*)/settings',
        name: 'repo-settings',
        meta: { requiresRepoAdmin: true },
        component: () => import('@/views/RepoSettingsView.vue'),
        props: true,
      },
      {
        path: ':folder/:name',
        name: 'repo',
        meta: { publicRead: true },
        beforeEnter: redirectReservedRepoTab,
        component: () => import('@/views/RepoHomeView.vue'),
        props: (route) => ({
          repoPath: `${String(route.params.folder)}/${String(route.params.name)}`,
        }),
      },
      {
        path: ':folderPath',
        name: 'folder',
        meta: { publicRead: true },
        component: () => import('@/views/PathEntryView.vue'),
        props: true,
      },
      {
        path: ':pathMatch(.*)*',
        name: 'not-found',
        meta: { publicRead: true },
        component: () => import('@/views/NotFoundView.vue'),
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
