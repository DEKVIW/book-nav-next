<script setup lang="ts">
/**
 * Portal shell — mecha hangar HUD
 * Desktop: always-visible rail, expanded | collapsed (icon-only)
 * Mobile: overlay drawer
 */
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { useAuthStore } from '@/shared/stores/auth'
import { usePortalStore } from '@/shared/stores/portal'
import AppIcon from '@/shared/ui/AppIcon.vue'
import { iconForCategory } from '@/shared/icons/registry'
import type { Category } from '@/shared/types/models'

const props = withDefaults(
  defineProps<{
    categories?: Category[]
    aiAvailable?: boolean
    useAi?: boolean
  }>(),
  { aiAvailable: false, useAi: false },
)

const emit = defineEmits<{
  selectSubcategory: [parentId: number, childId: number | 'root']
  search: [q: string]
  'update:useAi': [v: boolean]
}>()

const auth = useAuthStore()
const portal = usePortalStore()

const STORAGE_MODE = 'booknav_sidebar_mode'
const STORAGE_LEGACY = 'booknav_sidebar_open'
const STORAGE_SUBMENU = 'booknav_submenu_'

/** Desktop: expanded labels | collapsed icon-only. Mobile open uses expanded labels in drawer. */
const railExpanded = ref(true)
const mobileOpen = ref(false)
const searchInput = ref('')
const activeCat = ref<number | null>(null)
const expanded = ref<Set<number>>(new Set())
const flyoutId = ref<number | null>(null)
/** 点击/编程滚动期间锁定选中态，避免 IntersectionObserver 在 smooth scroll 途中乱跳 */
const activeLocked = ref(false)
let unlockTimer: ReturnType<typeof setTimeout> | null = null
let scrollEndHandler: (() => void) | null = null

const navCats = computed(() => props.categories || portal.categories)
const aiOn = computed({
  get: () => props.useAi,
  set: (v: boolean) => emit('update:useAi', v),
})

const isMobile = () => typeof window !== 'undefined' && window.innerWidth < 900

function loadSidebarState() {
  const mode = localStorage.getItem(STORAGE_MODE)
  if (mode === 'collapsed' || mode === 'expanded') {
    railExpanded.value = mode === 'expanded'
    return
  }
  // migrate legacy: open=true → expanded, else collapsed icon rail
  const legacy = localStorage.getItem(STORAGE_LEGACY)
  railExpanded.value = legacy === 'true'
}

function persistMode() {
  localStorage.setItem(STORAGE_MODE, railExpanded.value ? 'expanded' : 'collapsed')
  localStorage.setItem(STORAGE_LEGACY, railExpanded.value ? 'true' : 'false')
}

function toggleRail() {
  if (isMobile()) {
    mobileOpen.value = !mobileOpen.value
    return
  }
  railExpanded.value = !railExpanded.value
  flyoutId.value = null
  persistMode()
}

function closeMobile() {
  mobileOpen.value = false
  flyoutId.value = null
}

function onOverlayClick() {
  if (isMobile()) closeMobile()
}

function toggleSubmenu(cat: Category, e?: Event) {
  e?.stopPropagation()
  e?.preventDefault()
  if (!railExpanded.value && !isMobile()) {
    // collapsed: toggle flyout
    flyoutId.value = flyoutId.value === cat.id ? null : cat.id
    return
  }
  const next = new Set(expanded.value)
  if (next.has(cat.id)) next.delete(cat.id)
  else next.add(cat.id)
  expanded.value = next
  localStorage.setItem(STORAGE_SUBMENU + cat.id, next.has(cat.id) ? 'true' : 'false')
}

function isSubOpen(id: number) {
  if (!railExpanded.value && !isMobile()) return flyoutId.value === id
  return expanded.value.has(id)
}

function restoreSubmenus() {
  const set = new Set<number>()
  for (const cat of navCats.value) {
    if (localStorage.getItem(STORAGE_SUBMENU + cat.id) === 'true') set.add(cat.id)
  }
  expanded.value = set
}

function lockActive(id: number) {
  activeCat.value = id
  activeLocked.value = true
  if (unlockTimer) {
    clearTimeout(unlockTimer)
    unlockTimer = null
  }
  if (scrollEndHandler) {
    window.removeEventListener('scrollend', scrollEndHandler)
    scrollEndHandler = null
  }
  // smooth scroll 结束后再交给 observer；scrollend 不支持时用超时兜底
  scrollEndHandler = () => {
    activeLocked.value = false
    if (unlockTimer) {
      clearTimeout(unlockTimer)
      unlockTimer = null
    }
    if (scrollEndHandler) {
      window.removeEventListener('scrollend', scrollEndHandler)
      scrollEndHandler = null
    }
  }
  window.addEventListener('scrollend', scrollEndHandler, { once: true })
  unlockTimer = setTimeout(() => {
    activeLocked.value = false
    unlockTimer = null
    if (scrollEndHandler) {
      window.removeEventListener('scrollend', scrollEndHandler)
      scrollEndHandler = null
    }
  }, 900)
}

function scrollToCategory(id: number, opts?: { closeOnMobile?: boolean }) {
  if (portal.searchResults) {
    portal.clearSearch()
    searchInput.value = ''
  }
  lockActive(id)
  const el = document.getElementById(`cat-${id}`)
  if (el) {
    const top = el.getBoundingClientRect().top + window.scrollY - 72
    window.scrollTo({ top, behavior: 'smooth' })
    history.replaceState(null, '', `#cat-${id}`)
  }
  if (opts?.closeOnMobile !== false && isMobile()) closeMobile()
}

function onRootClick(cat: Category) {
  scrollToCategory(cat.id)
  flyoutId.value = null
}

function onChildClick(parent: Category, child: Category) {
  scrollToCategory(parent.id)
  emit('selectSubcategory', parent.id, child.id)
  flyoutId.value = null
  if (isMobile()) closeMobile()
}

function onRootAsUncategorized(cat: Category) {
  scrollToCategory(cat.id)
  emit('selectSubcategory', cat.id, 'root')
  flyoutId.value = null
  if (isMobile()) closeMobile()
}

function onSearch(e: Event) {
  e.preventDefault()
  const q = searchInput.value.trim()
  if (!q) {
    portal.clearSearch()
    return
  }
  emit('search', q)
}

function clearSearch() {
  searchInput.value = ''
  portal.clearSearch()
}

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (flyoutId.value != null) flyoutId.value = null
    else if (mobileOpen.value) closeMobile()
    else if (portal.searchResults) clearSearch()
  }
}

function onDocClick(e: MouseEvent) {
  if (flyoutId.value == null) return
  const t = e.target as HTMLElement | null
  if (t?.closest?.('.side-group')) return
  flyoutId.value = null
}

let io: IntersectionObserver | null = null

function pickActiveFromViewport() {
  if (activeLocked.value) return
  const sections = Array.from(document.querySelectorAll<HTMLElement>('[id^="cat-"]'))
  if (!sections.length) return
  // 顶栏下方「阅读线」：取已越过该线的 section 中 top 最大者（当前正在读的舱段）
  const line = 72 + 24
  let chosen: HTMLElement | null = null
  let bestTop = -Infinity
  for (const el of sections) {
    const top = el.getBoundingClientRect().top
    if (top <= line && top > bestTop) {
      bestTop = top
      chosen = el
    }
  }
  if (!chosen) {
    let minTop = Infinity
    for (const el of sections) {
      const top = el.getBoundingClientRect().top
      if (top < minTop) {
        minTop = top
        chosen = el
      }
    }
  }
  if (!chosen) return
  const id = Number(chosen.id.replace('cat-', ''))
  if (!Number.isNaN(id) && activeCat.value !== id) {
    activeCat.value = id
  }
}

function bindObserver() {
  io?.disconnect()
  io = new IntersectionObserver(
    () => {
      pickActiveFromViewport()
    },
    {
      // 宽观察带，具体选中由 pickActiveFromViewport 计算
      rootMargin: '-10% 0px -40% 0px',
      threshold: [0, 0.05, 0.15, 0.3, 0.5],
    },
  )
  document.querySelectorAll('[id^="cat-"]').forEach((el) => io?.observe(el))
}

watch(
  () => navCats.value.map((c) => c.id).join(','),
  () => {
    restoreSubmenus()
    requestAnimationFrame(() => setTimeout(bindObserver, 50))
  },
)

onMounted(() => {
  loadSidebarState()
  restoreSubmenus()
  window.addEventListener('keydown', onKey)
  document.addEventListener('click', onDocClick)
  setTimeout(bindObserver, 200)
  if (location.hash.startsWith('#cat-')) {
    const id = Number(location.hash.replace('#cat-', ''))
    if (!Number.isNaN(id)) {
      setTimeout(() => scrollToCategory(id, { closeOnMobile: false }), 100)
    }
  }
})

onUnmounted(() => {
  window.removeEventListener('keydown', onKey)
  document.removeEventListener('click', onDocClick)
  if (scrollEndHandler) window.removeEventListener('scrollend', scrollEndHandler)
  if (unlockTimer) clearTimeout(unlockTimer)
  io?.disconnect()
})

defineExpose({
  openSidebar: () => {
    if (isMobile()) mobileOpen.value = true
    else {
      railExpanded.value = true
      persistMode()
    }
  },
  closeSidebar: () => {
    if (isMobile()) closeMobile()
    else {
      railExpanded.value = false
      persistMode()
    }
  },
  scrollToCategory,
})
</script>

<template>
  <div
    class="shell"
    :class="{
      'shell--rail-expanded': railExpanded,
      'shell--rail-collapsed': !railExpanded,
      'shell--mobile-open': mobileOpen,
    }"
  >
    <header class="top-rail">
      <!-- 仅移动端：打开侧栏抽屉。桌面用侧栏底部折叠钮，不占顶栏 -->
      <button
        type="button"
        class="m-btn m-btn--ghost m-btn--icon top-rail__menu"
        :aria-label="mobileOpen ? '关闭菜单' : '打开菜单'"
        :aria-expanded="mobileOpen"
        @click="toggleRail"
      >
        <AppIcon :name="mobileOpen ? 'x' : 'menu'" :size="18" />
      </button>

      <form class="search-slot" @submit="onSearch">
        <div class="radar-search">
          <AppIcon name="search" :size="16" class="radar-search__icon" />
          <input
            v-model="searchInput"
            type="search"
            class="radar-search__input"
            placeholder="搜索网站标题、描述或链接…"
            enterkeyhint="search"
          />
          <button v-if="searchInput" type="button" class="clear" aria-label="清除" @click="clearSearch">
            <AppIcon name="x" :size="16" />
          </button>
        </div>
      </form>

      <div class="top-rail__actions">
        <label v-if="aiAvailable" class="ai-toggle" title="启用 AI 智能搜索">
          <input v-model="aiOn" type="checkbox" />
          <span>AI</span>
        </label>
        <template v-if="auth.isLoggedIn">
          <span class="user-chip">
            <span class="status-dot status-dot--ok" />
            {{ auth.user?.username }}
          </span>
          <RouterLink v-if="auth.isAdmin" class="m-btn m-btn--primary" to="/admin">管理后台</RouterLink>
          <button type="button" class="m-btn m-btn--ghost" @click="auth.logout()">退出</button>
        </template>
        <template v-else>
          <RouterLink class="m-btn m-btn--ghost" to="/login">登录</RouterLink>
        </template>
      </div>
      <div class="scan-edge" aria-hidden="true" />
    </header>

    <div
      class="sidebar-overlay"
      :class="{ 'sidebar-overlay--show': mobileOpen }"
      aria-hidden="true"
      @click="onOverlayClick"
    />

    <aside class="side-rail" aria-label="分类导航">
      <!-- 站点品牌：折叠仅图标，展开图标+站名（与分类项同一节奏） -->
      <div class="side-rail__head">
        <RouterLink
          to="/"
          class="side-brand"
          :title="portal.settings?.site_name || 'BookNav'"
          @click="clearSearch"
        >
          <span class="side-brand__logo">
            <img
              v-if="portal.settings?.site_logo || portal.settings?.site_favicon"
              :src="portal.settings?.site_logo || portal.settings?.site_favicon"
              alt=""
              class="side-brand__img"
            />
            <AppIcon v-else name="cpu" :size="20" />
          </span>
          <span class="side-brand__text">{{ portal.settings?.site_name || 'BookNav' }}</span>
        </RouterLink>
        <button
          type="button"
          class="side-rail__close side-rail__close--mobile"
          aria-label="关闭"
          @click="closeMobile"
        >
          <AppIcon name="x" :size="18" />
        </button>
      </div>

      <nav class="side-rail__nav">
        <div
          v-for="cat in navCats"
          :key="cat.id"
          class="side-group"
          :class="{ 'side-group--flyout': flyoutId === cat.id }"
        >
          <div
            class="side-item"
            :class="{
              'side-item--active': activeCat === cat.id,
              'side-item--has-children': !!cat.children?.length,
            }"
          >
            <button
              type="button"
              class="side-item__main"
              :title="cat.name"
              @click="onRootClick(cat)"
            >
              <span class="side-item__icon">
                <AppIcon :name="iconForCategory(cat.icon, cat.id)" :size="18" />
              </span>
              <span class="side-item__label">{{ cat.name }}</span>
              <span class="side-item__count">
                {{ cat.total_count_with_children ?? cat.website_count ?? cat.websites?.length ?? 0 }}
              </span>
            </button>
            <button
              v-if="cat.children?.length"
              type="button"
              class="side-item__toggle"
              :aria-expanded="isSubOpen(cat.id)"
              :aria-label="isSubOpen(cat.id) ? '收起子分类' : '展开子分类'"
              :title="cat.name"
              @click="toggleSubmenu(cat, $event)"
            >
              <AppIcon
                :name="railExpanded || isMobile() ? 'chevron-down' : 'chevron-right'"
                :size="14"
                class="side-item__chev"
                :class="{ 'side-item__chev--open': isSubOpen(cat.id) && (railExpanded || isMobile()) }"
              />
            </button>
          </div>

          <!-- expanded / mobile inline sub -->
          <div
            v-if="cat.children?.length && isSubOpen(cat.id) && (railExpanded || isMobile())"
            class="side-sub"
          >
            <button type="button" class="side-sub__item" @click="onRootAsUncategorized(cat)">
              <span>未分类</span>
              <span>{{ cat.direct_count ?? 0 }}</span>
            </button>
            <button
              v-for="ch in cat.children"
              :key="ch.id"
              type="button"
              class="side-sub__item"
              @click="onChildClick(cat, ch)"
            >
              <span>{{ ch.name }}</span>
              <span>{{ ch.website_count ?? 0 }}</span>
            </button>
          </div>

          <!-- collapsed flyout -->
          <div
            v-if="cat.children?.length && flyoutId === cat.id && !railExpanded && !isMobile()"
            class="side-flyout hull"
          >
            <div class="side-flyout__title">{{ cat.name }}</div>
            <button type="button" class="side-sub__item" @click="onRootAsUncategorized(cat)">
              <span>未分类</span>
              <span>{{ cat.direct_count ?? 0 }}</span>
            </button>
            <button
              v-for="ch in cat.children"
              :key="ch.id"
              type="button"
              class="side-sub__item"
              @click="onChildClick(cat, ch)"
            >
              <span>{{ ch.name }}</span>
              <span>{{ ch.website_count ?? 0 }}</span>
            </button>
          </div>
        </div>
      </nav>

      <div class="side-rail__foot">
        <button
          type="button"
          class="side-rail__collapse"
          :title="railExpanded ? '折叠为图标' : '展开侧栏'"
          :aria-label="railExpanded ? '折叠为图标' : '展开侧栏'"
          @click="
            () => {
              if (isMobile()) closeMobile()
              else {
                railExpanded = !railExpanded
                flyoutId = null
                persistMode()
              }
            }
          "
        >
          <AppIcon :name="railExpanded || isMobile() ? 'chevrons-left' : 'chevrons-right'" :size="18" />
          <span class="side-rail__collapse-label">{{ railExpanded ? '收起' : '展开' }}</span>
        </button>
      </div>
    </aside>

    <main class="viewport">
      <slot />
    </main>
  </div>
</template>

<style scoped>
.shell {
  min-height: 100vh;
  /* desktop default: collapsed icon rail width */
  padding-left: var(--sidebar-width-collapsed);
  transition: padding-left var(--dur-base) var(--ease-hydraulic);
}
.shell--rail-expanded {
  padding-left: var(--sidebar-width);
}

.top-rail {
  position: sticky;
  top: 0;
  z-index: var(--z-rail);
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: 0 var(--space-4);
  height: var(--top-rail-height);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.04), transparent 60%),
    color-mix(in srgb, var(--bg-hull) 92%, transparent);
  backdrop-filter: blur(16px) saturate(1.25);
  border-bottom: 1px solid var(--stroke-dim);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}

/* 桌面：侧栏底栏折叠即可，顶栏不放汉堡，避免左侧空一块 */
.top-rail__menu {
  display: none;
  flex-shrink: 0;
}

.search-slot {
  flex: 1;
  max-width: 560px;
  margin: 0 auto 0 0;
}

@media (min-width: 900px) {
  .search-slot {
    max-width: 640px;
  }
}
.radar-search {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  height: 42px;
  padding: 0 14px;
  background: var(--bg-input);
  border: 1px solid var(--stroke-dim);
  box-shadow: inset 0 1px 2px rgba(0, 0, 0, 0.45);
  clip-path: polygon(8px 0, 100% 0, 100% calc(100% - 8px), calc(100% - 8px) 100%, 0 100%, 0 8px);
  transition: border-color var(--dur-fast), box-shadow var(--dur-fast);
}
.radar-search:focus-within {
  border-color: var(--energy);
  box-shadow:
    inset 0 1px 2px rgba(0, 0, 0, 0.4),
    0 0 0 1px var(--energy-dim),
    0 0 20px var(--energy-glow);
}
.radar-search__icon {
  color: var(--energy);
  opacity: 0.85;
  flex-shrink: 0;
}
.radar-search__input {
  flex: 1;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--text-primary);
  font: inherit;
  font-size: var(--text-sm);
  min-width: 0;
}
.clear {
  border: 0;
  background: transparent;
  color: var(--text-muted);
  display: grid;
  place-items: center;
  padding: 2px;
  border-radius: 4px;
}
.clear:hover {
  color: var(--text-primary);
}

.top-rail__actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-left: auto;
}
.user-chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: var(--text-xs);
  color: var(--text-secondary);
  font-family: var(--font-mono);
  padding: 4px 10px;
  border: 1px solid var(--stroke-dim);
  background: var(--bg-inset);
  clip-path: polygon(4px 0, 100% 0, 100% calc(100% - 4px), calc(100% - 4px) 100%, 0 100%, 0 4px);
}
.ai-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-family: var(--font-display);
  letter-spacing: 0.08em;
  color: var(--text-secondary);
  padding: 4px 8px;
  border: 1px solid var(--stroke-dim);
  background: var(--bg-inset);
  user-select: none;
}
.ai-toggle:has(input:checked) {
  color: var(--magenta);
  border-color: color-mix(in srgb, var(--magenta) 50%, var(--stroke-dim));
  box-shadow: 0 0 12px rgba(179, 136, 255, 0.2);
}
.ai-toggle input {
  accent-color: var(--magenta);
}

/* —— Icon rail：通顶，填满左上角（顶栏只在内容区，侧栏占满左侧全高） —— */
.side-rail {
  position: fixed;
  z-index: var(--z-sidebar);
  top: 0;
  left: 0;
  bottom: 0;
  width: var(--sidebar-width-collapsed);
  border-right: 1px solid var(--stroke-dim);
  background:
    linear-gradient(180deg, rgba(61, 231, 255, 0.04), transparent 28%),
    linear-gradient(90deg, rgba(61, 231, 255, 0.03), transparent 50%),
    color-mix(in srgb, var(--bg-hull) 97%, transparent);
  backdrop-filter: blur(16px);
  display: flex;
  flex-direction: column;
  overflow: visible;
  transition: width var(--dur-base) var(--ease-hydraulic);
  box-shadow: 8px 0 32px rgba(0, 0, 0, 0.35);
}
.shell--rail-expanded .side-rail {
  width: var(--sidebar-width);
  overflow: hidden;
}

/* 品牌行与顶栏同高，视觉上连成一体 */
.side-rail__head {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  height: var(--top-rail-height);
  min-height: var(--top-rail-height);
  max-height: var(--top-rail-height);
  padding: 0 10px;
  border-bottom: 1px solid var(--stroke-dim);
  justify-content: center;
  box-sizing: border-box;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.04), transparent 60%),
    color-mix(in srgb, var(--bg-hull) 97%, transparent);
}
.shell--rail-expanded .side-rail__head {
  justify-content: flex-start;
  padding: 0 12px;
}

/* 站点品牌行 —— 折叠只 logo，展开 logo + 站名 */
.side-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  flex: 1;
  text-decoration: none;
  color: var(--text-primary);
  border-radius: 12px;
  padding: 4px;
  transition: background 0.12s;
}
.shell--rail-collapsed .side-brand {
  flex: 0;
  justify-content: center;
  padding: 4px 0;
}
.side-brand:hover {
  background: rgba(61, 231, 255, 0.06);
}
.side-brand__logo {
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  color: var(--bg-void);
  background: linear-gradient(145deg, #a8f8ff 0%, var(--energy) 45%, #1a7a98 100%);
  border-radius: 12px;
  overflow: hidden;
  box-shadow:
    0 0 0 1px rgba(61, 231, 255, 0.35),
    0 0 18px rgba(61, 231, 255, 0.22);
}
.side-brand__img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.side-brand__text {
  font-family: var(--font-display);
  font-weight: 700;
  letter-spacing: 0.06em;
  font-size: 0.9rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
  opacity: 0;
  width: 0;
  max-width: 0;
  transition:
    opacity var(--dur-fast),
    width var(--dur-base),
    max-width var(--dur-base);
}
.shell--rail-expanded .side-brand__text {
  opacity: 1;
  width: auto;
  max-width: 160px;
}

.side-rail__close {
  border: 0;
  background: transparent;
  color: var(--text-muted);
  display: none;
  place-items: center;
  padding: 4px;
  flex-shrink: 0;
}
.side-rail__close:hover {
  color: var(--text-primary);
}

.side-rail__nav {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: visible;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 10px 8px;
  -webkit-overflow-scrolling: touch;
}

.side-rail__foot {
  flex-shrink: 0;
  border-top: 1px solid var(--stroke-dim);
  padding: 10px 8px 12px;
  display: flex;
  justify-content: center;
}
.side-rail__collapse {
  width: 100%;
  max-width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 44px;
  border: 1px solid var(--stroke-dim);
  border-radius: 12px;
  background: var(--bg-inset);
  color: var(--text-secondary);
  font: inherit;
  font-size: var(--text-sm);
  transition:
    background 0.15s,
    color 0.15s,
    border-color 0.15s,
    box-shadow 0.15s;
}
.side-rail__collapse:hover {
  color: var(--text-primary);
  border-color: rgba(61, 231, 255, 0.35);
  box-shadow: 0 0 14px rgba(61, 231, 255, 0.12);
}
.shell--rail-collapsed .side-rail__collapse {
  width: 44px;
  height: 44px;
  max-width: 44px;
  border-radius: 12px;
  padding: 0;
}
.side-rail__collapse-label {
  display: none;
}
.shell--rail-expanded .side-rail__collapse-label {
  display: inline;
}

.side-group {
  position: relative;
}

.side-item {
  display: flex;
  align-items: stretch;
  border: 1px solid transparent;
  border-radius: 12px;
  background: transparent;
  transition: background-color 0.12s ease, border-color 0.12s ease;
}
.side-item--active {
  background: rgba(61, 231, 255, 0.1);
  border-color: rgba(61, 231, 255, 0.18);
}
.shell--rail-collapsed .side-item--active {
  background: rgba(61, 231, 255, 0.14);
  border-color: transparent;
  border-radius: 14px;
}

.side-item__main {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 8px;
  border: 0;
  background: transparent;
  color: var(--text-secondary);
  font: inherit;
  text-align: left;
}
.shell--rail-collapsed .side-item__main {
  justify-content: center;
  padding: 10px 0;
}
.side-item--active .side-item__main,
.side-item__main:hover {
  color: var(--text-primary);
}

.side-item__icon {
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  color: var(--text-secondary);
  border-radius: 10px;
  transition:
    color 0.12s,
    background 0.12s,
    box-shadow 0.12s;
}
.side-item--active .side-item__icon,
.side-item__main:hover .side-item__icon {
  color: var(--energy);
  background: rgba(61, 231, 255, 0.12);
  box-shadow: 0 0 12px rgba(61, 231, 255, 0.15);
}
.shell--rail-collapsed .side-item--active .side-item__icon {
  background: rgba(61, 231, 255, 0.16);
  color: var(--energy);
}

.side-item__label {
  flex: 1;
  font-size: var(--text-sm);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.side-item__count {
  font-family: var(--font-mono);
  font-size: var(--text-xs);
  color: var(--text-muted);
}
.shell--rail-collapsed .side-item__label,
.shell--rail-collapsed .side-item__count {
  display: none;
}

.side-item__toggle {
  width: 28px;
  border: 0;
  background: transparent;
  color: var(--text-muted);
  display: grid;
  place-items: center;
  flex-shrink: 0;
}
.shell--rail-collapsed .side-item__toggle {
  display: none;
}
.side-item__toggle:hover {
  color: var(--energy);
}
.side-item__chev {
  transition: transform var(--dur-fast);
}
.side-item__chev--open {
  transform: rotate(180deg);
}

.side-sub {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 2px 0 8px 14px;
}
.side-sub__item {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  border: 0;
  background: transparent;
  color: var(--text-muted);
  font: inherit;
  font-size: var(--text-xs);
  text-align: left;
  padding: 8px 10px;
  border-radius: 8px;
}
.side-sub__item:hover {
  color: var(--text-primary);
  background: var(--bg-panel);
}
.side-sub__item span:last-child {
  font-family: var(--font-mono);
  opacity: 0.8;
}

.side-flyout {
  position: absolute;
  left: calc(100% + 8px);
  top: 0;
  z-index: 80;
  min-width: 180px;
  max-width: 240px;
  padding: 8px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.55);
}
.side-flyout__title {
  font-size: 11px;
  font-family: var(--font-display);
  letter-spacing: 0.08em;
  color: var(--energy);
  padding: 6px 10px 8px;
  border-bottom: 1px solid var(--stroke-dim);
  margin-bottom: 4px;
}

.viewport {
  min-width: 0;
  padding: var(--space-6);
  max-width: var(--content-max);
  width: 100%;
  margin: 0 auto;
  box-sizing: border-box;
}

.sidebar-overlay {
  position: fixed;
  inset: 0;
  z-index: 50;
  background: rgba(0, 0, 0, 0.55);
  opacity: 0;
  pointer-events: none;
  transition: opacity var(--dur-base);
}
.sidebar-overlay--show {
  opacity: 1;
  pointer-events: auto;
}

@media (min-width: 900px) {
  .sidebar-overlay,
  .sidebar-overlay--show {
    display: none !important;
  }
  .side-rail__close--mobile {
    display: none !important;
  }
}

@media (max-width: 899px) {
  .top-rail__menu {
    display: inline-flex;
  }
  .shell,
  .shell--rail-expanded,
  .shell--rail-collapsed {
    padding-left: 0;
  }
  .side-rail {
    width: min(var(--sidebar-width), 86vw) !important;
    transform: translateX(-105%);
    transition: transform var(--dur-base) var(--ease-hydraulic);
    overflow: hidden;
  }
  .shell--mobile-open .side-rail {
    transform: translateX(0);
  }
  .side-brand__text {
    opacity: 1 !important;
    width: auto !important;
    max-width: 160px !important;
  }
  .side-rail__head {
    justify-content: flex-start !important;
    padding: 0 12px !important;
  }
  .side-item__label,
  .side-item__count {
    display: initial !important;
  }
  .side-item__toggle {
    display: grid !important;
  }
  .side-rail__collapse-label {
    display: inline !important;
  }
  .side-rail__collapse {
    width: 100% !important;
    max-width: none !important;
    height: auto !important;
  }
  .side-rail__close--mobile {
    display: grid;
  }
  .side-flyout {
    display: none !important;
  }
  .search-slot {
    max-width: none;
    flex: 1;
    min-width: 0;
  }
  .viewport {
    padding: var(--space-4) var(--space-3) var(--space-8);
  }
  .top-rail {
    gap: var(--space-2);
    padding: 0 var(--space-3);
  }
  .user-chip {
    display: none;
  }
}

@media (max-width: 480px) {
  .ai-toggle span {
    display: none;
  }
  .top-rail__actions .m-btn {
    padding-inline: 10px;
  }
}
</style>
