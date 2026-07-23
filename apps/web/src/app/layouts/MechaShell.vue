<script setup lang="ts">
/**
 * 侧栏行为对齐旧版：
 * - 默认收起，汉堡切换（桌面+移动均有效）
 * - localStorage 记忆开关
 * - 一级锚点滚动；仅移动端点击后关闭
 * - 二级子菜单展开/收起 + 状态记忆
 * - 点击子类：滚动父舱段 + 通知主区切换 Tab
 */
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { useAuthStore } from '@/shared/stores/auth'
import { usePortalStore } from '@/shared/stores/portal'
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

const STORAGE_SIDEBAR = 'booknav_sidebar_open'
const STORAGE_SUBMENU = 'booknav_submenu_'

const sidebarOpen = ref(false)
const searchInput = ref('')
const activeCat = ref<number | null>(null)
const expanded = ref<Set<number>>(new Set())

const navCats = computed(() => props.categories || portal.categories)
const aiOn = computed({
  get: () => props.useAi,
  set: (v: boolean) => emit('update:useAi', v),
})

const isMobile = () => window.innerWidth < 900

function loadSidebarState() {
  const saved = localStorage.getItem(STORAGE_SIDEBAR)
  // 旧版默认关闭；仅明确 true 时打开
  sidebarOpen.value = saved === 'true'
}

function persistSidebar() {
  localStorage.setItem(STORAGE_SIDEBAR, sidebarOpen.value ? 'true' : 'false')
}

function toggleSidebar() {
  sidebarOpen.value = !sidebarOpen.value
  persistSidebar()
}

function closeSidebar() {
  if (!sidebarOpen.value) return
  sidebarOpen.value = false
  persistSidebar()
}

/** 仅移动端点遮罩关闭；桌面无遮罩交互 */
function onOverlayClick() {
  if (isMobile()) closeSidebar()
}

function openSidebar() {
  sidebarOpen.value = true
  persistSidebar()
}

function toggleSubmenu(cat: Category, e?: Event) {
  e?.stopPropagation()
  e?.preventDefault()
  const next = new Set(expanded.value)
  if (next.has(cat.id)) next.delete(cat.id)
  else next.add(cat.id)
  expanded.value = next
  localStorage.setItem(STORAGE_SUBMENU + cat.id, next.has(cat.id) ? 'true' : 'false')
}

function isExpanded(id: number) {
  return expanded.value.has(id)
}

function restoreSubmenus() {
  const set = new Set<number>()
  for (const cat of navCats.value) {
    if (localStorage.getItem(STORAGE_SUBMENU + cat.id) === 'true') {
      set.add(cat.id)
    }
  }
  expanded.value = set
}

function scrollToCategory(id: number, opts?: { closeOnMobile?: boolean }) {
  // 搜索中先退出搜索层
  if (portal.searchResults) {
    portal.clearSearch()
    searchInput.value = ''
  }
  const el = document.getElementById(`cat-${id}`)
  if (el) {
    const top = el.getBoundingClientRect().top + window.scrollY - 72
    window.scrollTo({ top, behavior: 'smooth' })
    history.replaceState(null, '', `#cat-${id}`)
    activeCat.value = id
  }
  // 对齐旧版：仅移动端点击导航后关闭侧栏
  if (opts?.closeOnMobile !== false && isMobile()) {
    closeSidebar()
  }
}

function onRootClick(cat: Category) {
  // 对齐旧版：点文字 = 定位；展开只靠箭头
  scrollToCategory(cat.id)
}

function onChildClick(parent: Category, child: Category) {
  scrollToCategory(parent.id)
  emit('selectSubcategory', parent.id, child.id)
  if (isMobile()) closeSidebar()
}

function onRootAsUncategorized(cat: Category) {
  scrollToCategory(cat.id)
  emit('selectSubcategory', cat.id, 'root')
  if (isMobile()) closeSidebar()
}

function onSearch(e: Event) {
  e.preventDefault()
  const q = searchInput.value.trim()
  if (!q) {
    portal.clearSearch()
    return
  }
  // 交给首页：支持 AI 开关
  emit('search', q)
}

function clearSearch() {
  searchInput.value = ''
  portal.clearSearch()
}

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (sidebarOpen.value) closeSidebar()
    else if (portal.searchResults) clearSearch()
  }
}

let io: IntersectionObserver | null = null

function bindObserver() {
  io?.disconnect()
  io = new IntersectionObserver(
    (entries) => {
      // 取最靠近视口上部的可见 section
      const visible = entries
        .filter((en) => en.isIntersecting)
        .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)
      if (visible[0]) {
        const id = visible[0].target.id.replace('cat-', '')
        activeCat.value = Number(id)
      }
    },
    { rootMargin: '-15% 0px -55% 0px', threshold: [0, 0.1, 0.3] },
  )
  document.querySelectorAll('[id^="cat-"]').forEach((el) => io?.observe(el))
}

watch(
  () => navCats.value.map((c) => c.id).join(','),
  () => {
    restoreSubmenus()
    // DOM 更新后再绑 observer
    requestAnimationFrame(() => setTimeout(bindObserver, 50))
  },
)

onMounted(() => {
  loadSidebarState()
  restoreSubmenus()
  window.addEventListener('keydown', onKey)
  setTimeout(bindObserver, 200)
  // hash 直达
  if (location.hash.startsWith('#cat-')) {
    const id = Number(location.hash.replace('#cat-', ''))
    if (!Number.isNaN(id)) {
      setTimeout(() => scrollToCategory(id, { closeOnMobile: false }), 100)
    }
  }
})

onUnmounted(() => {
  window.removeEventListener('keydown', onKey)
  io?.disconnect()
})

defineExpose({ openSidebar, closeSidebar, scrollToCategory })
</script>

<template>
  <div
    class="shell"
    :class="{
      'shell--sidebar-open': sidebarOpen,
    }"
  >
    <header class="top-rail">
      <button
        type="button"
        class="m-btn m-btn--ghost m-btn--icon"
        :aria-label="sidebarOpen ? '收起侧栏' : '展开侧栏'"
        :aria-expanded="sidebarOpen"
        @click="toggleSidebar"
      >
        <span class="icon-bars" :class="{ 'icon-bars--open': sidebarOpen }" />
      </button>

      <RouterLink to="/" class="brand" @click="clearSearch">
        <span class="brand__mark" aria-hidden="true" />
        <span class="brand__text">{{ portal.settings?.site_name || 'BookNav' }}</span>
      </RouterLink>

      <form class="search-slot" @submit="onSearch">
        <div class="radar-search">
          <span class="radar-search__glyph" aria-hidden="true" />
          <input
            v-model="searchInput"
            type="search"
            class="radar-search__input"
            placeholder="搜索网站标题、描述或链接…"
            enterkeyhint="search"
          />
          <button v-if="searchInput" type="button" class="clear" @click="clearSearch">×</button>
        </div>
      </form>

      <div class="top-rail__actions">
        <label v-if="aiAvailable" class="ai-toggle" title="启用 AI 智能搜索（需后台配置）">
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

    <!-- 遮罩仅移动端：点遮罩关闭。桌面钉住侧栏时不盖主内容，避免误关 -->
    <div
      class="sidebar-overlay"
      :class="{ 'sidebar-overlay--show': sidebarOpen }"
      aria-hidden="true"
      @click="onOverlayClick"
    />

    <aside class="side-rail" aria-label="分类导航">
      <div class="side-rail__head">
        <span class="side-rail__title">分类</span>
        <button
          type="button"
          class="side-rail__close side-rail__close--mobile"
          aria-label="关闭"
          @click="closeSidebar"
        >
          ×
        </button>
      </div>

      <nav class="side-rail__nav">
        <div v-for="cat in navCats" :key="cat.id" class="side-group">
          <div
            class="side-item"
            :class="{
              'side-item--active': activeCat === cat.id,
              'side-item--has-children': !!cat.children?.length,
            }"
          >
            <button type="button" class="side-item__main" @click="onRootClick(cat)">
              <span
                class="side-item__bar"
                :style="cat.color ? { background: cat.color } : undefined"
              />
              <span class="side-item__label">{{ cat.name }}</span>
              <span class="side-item__count">
                {{ cat.total_count_with_children ?? cat.website_count ?? cat.websites?.length ?? 0 }}
              </span>
            </button>
            <button
              v-if="cat.children?.length"
              type="button"
              class="side-item__toggle"
              :aria-expanded="isExpanded(cat.id)"
              :aria-label="isExpanded(cat.id) ? '收起' : '展开'"
              @click="toggleSubmenu(cat, $event)"
            >
              <span class="chevron" :class="{ 'chevron--open': isExpanded(cat.id) }" />
            </button>
          </div>

          <div v-if="cat.children?.length && isExpanded(cat.id)" class="side-sub">
            <button type="button" class="side-sub__item" @click="onRootAsUncategorized(cat)">
              未分类
              <span>{{ cat.direct_count ?? 0 }}</span>
            </button>
            <button
              v-for="ch in cat.children"
              :key="ch.id"
              type="button"
              class="side-sub__item"
              @click="onChildClick(cat, ch)"
            >
              {{ ch.name }}
              <span>{{ ch.website_count ?? 0 }}</span>
            </button>
          </div>
        </div>
      </nav>

      <!-- 收起固定底部 -->
      <div class="side-rail__foot">
        <button type="button" class="side-rail__collapse" @click="closeSidebar">
          <span class="side-rail__collapse-icon" aria-hidden="true">«</span>
          <span>收起</span>
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
  /* 默认不占侧栏宽度，打开时通过 padding 让位（桌面） */
  padding-left: 0;
  transition: padding-left var(--dur-base) var(--ease-out);
}
.shell--sidebar-open {
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
    linear-gradient(180deg, rgba(255, 255, 255, 0.03), transparent 60%),
    color-mix(in srgb, var(--bg-hull) 94%, transparent);
  backdrop-filter: blur(14px) saturate(1.2);
  border-bottom: 1px solid var(--stroke-dim);
  box-shadow: 0 8px 28px rgba(0, 0, 0, 0.35);
}

.brand {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  text-decoration: none;
  color: var(--text-primary);
  flex-shrink: 0;
}
.brand__mark {
  width: 22px;
  height: 22px;
  background: linear-gradient(135deg, var(--glow-cyan), var(--glow-magenta));
  clip-path: polygon(20% 0, 100% 0, 80% 100%, 0 100%);
  box-shadow: var(--glow-sm);
}
.brand__text {
  font-family: var(--font-display);
  font-weight: 700;
  letter-spacing: 0.1em;
  font-size: var(--text-sm);
  max-width: 28vw;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.search-slot {
  flex: 1;
  max-width: 480px;
  margin: 0 auto;
}
.radar-search {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  height: 40px;
  padding: 0 12px;
  background: var(--bg-input);
  border: 1px solid var(--stroke-dim);
  box-shadow: inset 0 1px 2px rgba(0, 0, 0, 0.4);
  clip-path: polygon(8px 0, 100% 0, 100% calc(100% - 8px), calc(100% - 8px) 100%, 0 100%, 0 8px);
  transition: border-color var(--dur-fast), box-shadow var(--dur-fast);
}
.radar-search:focus-within {
  border-color: var(--energy);
  box-shadow:
    inset 0 1px 2px rgba(0, 0, 0, 0.4),
    0 0 0 1px var(--energy-dim),
    0 0 16px var(--energy-glow);
}
.radar-search__glyph {
  width: 14px;
  height: 14px;
  border: 2px solid var(--energy);
  border-radius: 50%;
  opacity: 0.75;
  position: relative;
  flex-shrink: 0;
}
.radar-search__glyph::after {
  content: '';
  position: absolute;
  right: -5px;
  bottom: -4px;
  width: 8px;
  height: 2px;
  background: var(--energy);
  transform: rotate(45deg);
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
  cursor: pointer;
  font-size: 18px;
  line-height: 1;
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
  cursor: pointer;
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

/* —— 侧栏：head 固定 + nav 滚动 + foot 收起固定 —— */
.side-rail {
  position: fixed;
  z-index: var(--z-sidebar);
  top: var(--top-rail-height);
  left: 0;
  bottom: 0;
  width: var(--sidebar-width);
  border-right: 1px solid var(--stroke-dim);
  background:
    linear-gradient(90deg, rgba(61, 231, 255, 0.03), transparent 40%),
    color-mix(in srgb, var(--bg-hull) 97%, transparent);
  backdrop-filter: blur(14px);
  padding: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  transform: translateX(-100%);
  transition: transform var(--dur-base) var(--ease-hydraulic);
  box-shadow: none;
}
.shell--sidebar-open .side-rail {
  transform: translateX(0);
  box-shadow: 12px 0 40px rgba(0, 0, 0, 0.45);
}

.side-rail__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
  min-height: 48px;
  padding: 0 var(--space-3);
  border-bottom: 1px solid var(--stroke-dim);
}
.side-rail__title {
  font-size: var(--text-sm);
  color: var(--text-primary);
  font-weight: 600;
  letter-spacing: 0.04em;
}
.side-rail__close {
  border: 0;
  background: transparent;
  color: var(--text-muted);
  font-size: 20px;
  cursor: pointer;
  line-height: 1;
  padding: 0 4px;
}
.side-rail__close:hover {
  color: var(--text-primary);
}
.side-rail__close--mobile {
  display: none;
}

.side-rail__nav {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: var(--space-2);
  -webkit-overflow-scrolling: touch;
}

.side-rail__foot {
  flex-shrink: 0;
  border-top: 1px solid var(--stroke-dim);
  padding: var(--space-2);
  background: color-mix(in srgb, var(--bg-hull) 98%, transparent);
}
.side-rail__collapse {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 40px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  font: inherit;
  font-size: var(--text-sm);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.side-rail__collapse:hover {
  color: var(--text-primary);
  background: var(--bg-panel);
}
.side-rail__collapse-icon {
  font-size: 14px;
  line-height: 1;
  opacity: 0.85;
}

.side-item {
  display: flex;
  align-items: stretch;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
}
.side-item--active {
  background: var(--bg-panel);
  border-color: var(--stroke-dim);
}
.side-item__main {
  flex: 1;
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-width: 0;
  padding: var(--space-2) var(--space-2) var(--space-2) var(--space-3);
  border: 0;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  font: inherit;
  text-align: left;
}
.side-item--active .side-item__main,
.side-item__main:hover {
  color: var(--text-primary);
}
.side-item__bar {
  width: 3px;
  height: 16px;
  flex-shrink: 0;
  background: transparent;
  border-radius: 1px;
}
.side-item--active .side-item__bar {
  background: var(--glow-cyan);
  box-shadow: var(--glow-sm);
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
.side-item__toggle {
  width: 32px;
  border: 0;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  display: grid;
  place-items: center;
  flex-shrink: 0;
}
.side-item__toggle:hover {
  color: var(--glow-cyan);
}
.chevron {
  width: 8px;
  height: 8px;
  border-right: 2px solid currentColor;
  border-bottom: 2px solid currentColor;
  transform: rotate(-45deg);
  transition: transform var(--dur-fast);
  margin-bottom: 3px;
}
.chevron--open {
  transform: rotate(45deg);
  margin-bottom: 0;
  margin-top: 3px;
}

.side-sub {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 2px 0 6px 18px;
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
  padding: 6px 8px;
  cursor: pointer;
  border-radius: var(--radius-sm);
}
.side-sub__item:hover {
  color: var(--text-primary);
  background: var(--bg-panel);
}
.side-sub__item span {
  font-family: var(--font-mono);
  opacity: 0.8;
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
  inset: var(--top-rail-height) 0 0 0;
  z-index: 50;
  background: rgba(0, 0, 0, 0.5);
  opacity: 0;
  pointer-events: none;
  transition: opacity var(--dur-base);
  cursor: pointer;
}
/* 默认：仅移动端侧栏打开时遮罩可点关 */
.sidebar-overlay--show {
  opacity: 1;
  pointer-events: auto;
}

/* 桌面：侧栏钉住，主内容可正常点选，绝不盖遮罩 */
@media (min-width: 900px) {
  .sidebar-overlay,
  .sidebar-overlay--show {
    display: none !important;
    pointer-events: none !important;
    opacity: 0 !important;
  }
}

.icon-bars {
  display: block;
  width: 16px;
  height: 2px;
  background: currentColor;
  box-shadow: 0 5px 0 currentColor, 0 -5px 0 currentColor;
  transition: box-shadow var(--dur-fast), transform var(--dur-fast);
}
.icon-bars--open {
  box-shadow: none;
  transform: rotate(45deg);
  position: relative;
}
.icon-bars--open::after {
  content: '';
  position: absolute;
  inset: 0;
  background: currentColor;
  transform: rotate(90deg);
}

/* 平板 */
@media (max-width: 1100px) and (min-width: 900px) {
  .viewport {
    padding: var(--space-4) var(--space-5);
  }
  .search-slot {
    max-width: 280px;
  }
}

/* 手机 / 小平板 */
@media (max-width: 899px) {
  .shell--sidebar-open {
    padding-left: 0;
  }
  .search-slot {
    max-width: none;
    flex: 1;
    min-width: 0;
  }
  .brand__text {
    display: none;
  }
  .side-rail {
    width: min(var(--sidebar-width), 86vw);
  }
  .side-rail__close--mobile {
    display: block;
  }
  /* 移动端底部用「收起」即可，顶栏汉堡负责打开 */
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
