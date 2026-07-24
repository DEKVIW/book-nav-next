import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/shared/stores/auth'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('@/modules/portal/pages/HomePage.vue'),
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('@/modules/portal/pages/LoginPage.vue'),
    },

    {
      path: '/goto/:id',
      name: 'goto',
      component: () => import('@/modules/portal/pages/GotoPage.vue'),
    },
    {
      path: '/admin',
      component: () => import('@/modules/admin/components/AdminLayout.vue'),
      meta: { requiresAdmin: true },
      children: [
        {
          path: '',
          name: 'admin',
          component: () => import('@/modules/admin/pages/DashboardPage.vue'),
        },
        {
          path: 'websites',
          name: 'admin-websites',
          component: () => import('@/modules/admin/pages/WebsitesPage.vue'),
        },
        {
          path: 'categories',
          name: 'admin-categories',
          component: () => import('@/modules/admin/pages/CategoriesPage.vue'),
        },
        {
          path: 'users',
          name: 'admin-users',
          component: () => import('@/modules/admin/pages/UsersPage.vue'),
          meta: { requiresSuper: true },
        },
        {
          path: 'logs',
          name: 'admin-logs',
          component: () => import('@/modules/admin/pages/LogsPage.vue'),
        },
        {
          path: 'settings',
          name: 'admin-settings',
          component: () => import('@/modules/admin/pages/SettingsPage.vue'),
          meta: { requiresSuper: true },
        },
        {
          // 死链检测并入「数据管理」Tab
          path: 'deadlinks',
          redirect: { path: '/admin/data', query: { tab: 'deadlinks' } },
        },
        {
          path: 'icons',
          name: 'admin-icons',
          component: () => import('@/modules/admin/pages/IconsPage.vue'),
          meta: { requiresSuper: true },
        },
        {
          path: 'data',
          name: 'admin-data',
          component: () => import('@/modules/admin/pages/DataPage.vue'),
          meta: { requiresSuper: true },
        },
        {
          path: 'backups',
          name: 'admin-backups',
          component: () => import('@/modules/admin/pages/BackupsPage.vue'),
          meta: { requiresSuper: true },
        },
        {
          path: 'jobs',
          name: 'admin-jobs',
          component: () => import('@/modules/admin/pages/JobsPage.vue'),
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/modules/portal/pages/NotFoundPage.vue'),
    },
  ],
  scrollBehavior(to, _from, saved) {
    if (saved) return saved
    if (to.hash) return { el: to.hash, behavior: 'smooth', top: 72 }
    return { top: 0 }
  },
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!auth.loaded) {
    await auth.fetchMe()
  }
  if (to.matched.some((r) => r.meta.requiresAdmin) && !auth.isAdmin) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.matched.some((r) => r.meta.requiresSuper) && !auth.isSuper) {
    return { name: 'admin' }
  }
  return true
})
