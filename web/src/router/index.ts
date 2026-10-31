import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { useRegistration } from '@/composables/useRegistration'
import { useSite } from '@/composables/useSite'

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
        path: 'settings/profile',
        name: 'profile',
        component: () => import('@/views/ProfileView.vue'),
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
        path: ':repoPath(.*)/tree',
        name: 'repo-tree',
        meta: { publicRead: true },
        component: () => import('@/views/RepoTreeView.vue'),
        props: true,
      },
      {
        path: ':repoPath(.*)/settings',
        name: 'repo-settings',
        component: () => import('@/views/RepoSettingsView.vue'),
        props: true,
      },
      {
        path: ':folder/:name',
        name: 'repo',
        meta: { publicRead: true },
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

export default router
