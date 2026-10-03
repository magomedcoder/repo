import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuth } from '@/composables/useAuth'

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
        component: () => import('@/views/HomeView.vue'),
      },
      {
        path: 'folders/:id',
        name: 'folder',
        component: () => import('@/views/FolderView.vue'),
        props: true,
      },
      {
        path: 'settings/keys',
        name: 'ssh-keys',
        component: () => import('@/views/SSHKeysView.vue'),
      },
      {
        path: ':owner/:repoPath(.*)/pulls/:number',
        name: 'repo-pull',
        component: () => import('@/views/PullView.vue'),
        props: true,
      },
      {
        path: ':owner/:repoPath(.*)/pulls',
        name: 'repo-pulls',
        component: () => import('@/views/PullsView.vue'),
        props: true,
      },
      {
        path: ':owner/:repoPath(.*)/issues/:number',
        name: 'repo-issue',
        component: () => import('@/views/IssueView.vue'),
        props: true,
      },
      {
        path: ':owner/:repoPath(.*)/issues',
        name: 'repo-issues',
        component: () => import('@/views/IssuesView.vue'),
        props: true,
      },
      {
        path: ':owner/:repoPath(.*)/commits/:sha',
        name: 'repo-commit',
        component: () => import('@/views/RepoCommitView.vue'),
        props: true,
      },
      {
        path: ':owner/:repoPath(.*)/commits',
        name: 'repo-commits',
        component: () => import('@/views/RepoCommitsView.vue'),
        props: true,
      },
      {
        path: ':owner/:repoPath(.*)/blob',
        name: 'repo-blob',
        component: () => import('@/views/RepoBlobView.vue'),
        props: true,
      },
      {
        path: ':owner/:repoPath(.*)/tree',
        name: 'repo-tree',
        component: () => import('@/views/RepoTreeView.vue'),
        props: true,
      },
      {
        path: ':owner/:repoPath(.*)/settings',
        name: 'repo-settings',
        component: () => import('@/views/RepoSettingsView.vue'),
        props: true,
      },
      {
        path: ':owner/:repoPath(.*)',
        name: 'repo',
        component: () => import('@/views/RepoHomeView.vue'),
        props: true,
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
  const auth = useAuth()
  if (!auth.ready.value) {
    await auth.fetchMe()
  }

  if (to.meta.requiresAuth && !auth.isAuthenticated.value) {
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

  return true
})

export default router
