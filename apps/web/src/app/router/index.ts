import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/shared/stores/auth'
import { usePortalStore } from '@/shared/stores/portal'
// Eager home: avoid extra round-trip after index.js (shell + cards paint sooner)
import HomePage from '@/modules/portal/pages/HomePage.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomePage,
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
        {
          path: 'about',
          name: 'admin-about',
          component: () => import('@/modules/admin/pages/AboutPage.vue'),
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
  const needsAdmin = to.matched.some((r) => r.meta.requiresAdmin)
  const needsSuper = to.matched.some((r) => r.meta.requiresSuper)
  // Portal: do not block first paint on /auth/me (~0.7s RTT). Admin still awaits session.
  if (!auth.loaded) {
    if (needsAdmin || needsSuper) {
      await auth.fetchMe()
    } else {
      void auth.fetchMe()
    }
  }
  if (needsAdmin && !auth.isAdmin) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (needsSuper && !auth.isSuper) {
    return { name: 'admin' }
  }
  return true
})

// Start home data fetch immediately after navigation (overlaps chunk/CSS paint)
router.afterEach((to) => {
  if (to.name === 'home') {
    void usePortalStore().loadHome()
  }
})
