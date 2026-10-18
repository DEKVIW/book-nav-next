<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/shared/stores/auth'
import AppIcon from '@/shared/ui/AppIcon.vue'
import ThemeMenu from '@/shared/ui/ThemeMenu.vue'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const COLLAPSE_KEY = 'booknav_admin_collapsed'
const collapsed = ref(localStorage.getItem(COLLAPSE_KEY) === '1')
const mobileOpen = ref(false)
const accountOpen = ref(false)
const avatarFailed = ref(false)

watch(collapsed, (v) => localStorage.setItem(COLLAPSE_KEY, v ? '1' : '0'))
watch(
  () => route.path,
  () => {
    mobileOpen.value = false
    accountOpen.value = false
  },
)

watch(
  () => auth.user?.avatar,
  () => {
    avatarFailed.value = false
  },
)

type NavItem = { to: string; label: string; icon: string; super?: boolean }

const menuItems = computed<NavItem[]>(() => [
  { to: '/admin/settings', label: '站点设置', icon: 'settings', super: true },
  { to: '/admin/categories', label: '分类管理', icon: 'folder' },
  { to: '/admin/websites', label: '链接管理', icon: 'globe' },
  { to: '/admin/users', label: '用户管理', icon: 'users', super: true },
  { to: '/admin/data', label: '数据管理', icon: 'server', super: true },
  { to: '/admin/icons', label: '图标管理', icon: 'image', super: true },
  { to: '/admin/backups', label: '备份管理', icon: 'archive', super: true },
  { to: '/admin/shares', label: '分享管理', icon: 'link' },
  { to: '/admin/about', label: '关于', icon: 'info' },
])

function visible(item: NavItem) {
  if (item.super) return auth.isSuper
  return true
}

function isActive(to: string) {
  if (to === '/admin') return route.path === '/admin'
  return route.path === to || route.path.startsWith(to + '/')
}

const pageTitle = computed(() => {
  const map: Record<string, string> = {
    '/admin': '仪表盘',
    '/admin/settings': '站点设置',
    '/admin/categories': '分类管理',
    '/admin/websites': '链接管理',
    '/admin/users': '用户管理',
    '/admin/data': '数据管理',
    '/admin/icons': '图标管理',
    '/admin/backups': '备份管理',
    '/admin/logs': '操作日志',
    '/admin/jobs': '任务中心',
    '/admin/shares': '分享管理',
    '/admin/about': '关于',
  }
  const path = route.path
  const keys = Object.keys(map).sort((a, b) => b.length - a.length)
  for (const k of keys) {
    if (path === k || path.startsWith(k + '/')) return map[k]
  }
  return '管理后台'
})

const initial = computed(() => (auth.user?.username || 'A').slice(0, 1).toUpperCase())
const accountTitle = computed(() => auth.user?.username || '管理员')

function toggleAccount() {
  accountOpen.value = !accountOpen.value
}

async function logout() {
  accountOpen.value = false
  await auth.logout()
  router.push('/login')
}

function onDocumentClick(event: MouseEvent) {
  const target = event.target as HTMLElement | null
  if (accountOpen.value && !target?.closest('.admin-account-slot')) accountOpen.value = false
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') accountOpen.value = false
}

onMounted(() => {
  document.addEventListener('click', onDocumentClick)
  document.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  document.removeEventListener('click', onDocumentClick)
  document.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <div
    class="admin-shell mecha-void"
    :class="{
      'admin-shell--collapsed': collapsed,
      'admin-shell--mobile-open': mobileOpen,
    }"
  >
    <header class="admin-topbar">
      <div class="admin-topbar__left">
        <button
          type="button"
          class="admin-icon-btn admin-icon-btn--mobile"
          aria-label="菜单"
          @click="mobileOpen = !mobileOpen"
        >
          <AppIcon name="menu" :size="20" />
        </button>
        <RouterLink to="/admin" class="admin-brand">后台管理</RouterLink>
        <span class="admin-topbar__sep">/</span>
        <span class="admin-topbar__crumb">{{ pageTitle }}</span>
      </div>
      <div class="admin-topbar__right">
        <ThemeMenu />
        <RouterLink class="admin-top-link" to="/" title="前台">
          <AppIcon name="home" :size="16" />
          <span class="admin-top-link__text">前台</span>
        </RouterLink>
        <div class="admin-account-slot">
          <button
            type="button"
            class="admin-account-trigger"
            :aria-label="accountTitle"
            :aria-expanded="accountOpen"
            aria-haspopup="menu"
            :title="accountTitle"
            @click.stop="toggleAccount"
          >
            <img
              v-if="auth.user?.avatar && !avatarFailed"
              class="admin-account-avatar admin-account-avatar--img"
              :src="auth.user.avatar"
              alt=""
              @error="avatarFailed = true"
            />
            <span v-else class="admin-account-avatar">{{ initial }}</span>
            <span class="admin-account-trigger__name">{{ auth.user?.username }}</span>
            <AppIcon name="chevron-down" :size="13" class="admin-account-trigger__chevron" />
          </button>

          <div v-if="accountOpen" class="admin-account-menu" role="menu">
            <div class="admin-account-menu__identity">
              <strong>{{ auth.user?.username }}</strong>
              <span>{{ auth.user?.email }}</span>
            </div>
            <div class="admin-account-menu__divider" />
            <RouterLink class="admin-account-menu__item" to="/" role="menuitem" @click="accountOpen = false">
              <AppIcon name="home" :size="16" />
              <span>返回前台</span>
            </RouterLink>
            <button type="button" class="admin-account-menu__item admin-account-menu__item--danger" role="menuitem" @click="logout">
              <AppIcon name="log-out" :size="16" />
              <span>退出登录</span>
            </button>
          </div>
        </div>
      </div>
    </header>

    <aside class="admin-sidebar">
      <div class="admin-sidebar__head">
        <RouterLink
          to="/admin"
          class="admin-nav-link admin-nav-link--dash"
          :class="{ active: isActive('/admin') }"
          title="仪表盘"
        >
          <AppIcon name="layout-dashboard" :size="18" />
          <span class="admin-nav-text">仪表盘</span>
        </RouterLink>
      </div>

      <nav class="admin-sidebar__nav">
        <RouterLink
          v-for="item in menuItems.filter(visible)"
          :key="item.to"
          :to="item.to"
          class="admin-nav-link"
          :class="{ active: isActive(item.to) }"
          :title="item.label"
        >
          <AppIcon :name="item.icon" :size="18" />
          <span class="admin-nav-text">{{ item.label }}</span>
        </RouterLink>
      </nav>

      <div class="admin-sidebar__foot">
        <button
          type="button"
          class="admin-sidebar__collapse"
          :title="collapsed ? '展开' : '收起'"
          @click="collapsed = !collapsed"
        >
          <AppIcon :name="collapsed ? 'chevrons-right' : 'chevrons-left'" :size="16" />
          <span class="admin-nav-text">{{ collapsed ? '展开' : '收起' }}</span>
        </button>
      </div>
    </aside>

    <div class="admin-overlay" @click="mobileOpen = false" />

    <main class="admin-main">
      <div class="admin-container">
        <RouterView />
      </div>
    </main>
  </div>
</template>
