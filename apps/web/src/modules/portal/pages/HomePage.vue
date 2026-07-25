<script setup lang="ts">
/**
 * 前台首页 — 功能对齐旧站 + 市面导航「就地展开」
 *
 * 看全部：display_limit 限显 →「展开全部」懒加载该分类/子类全部链接（不跳独立页）
 * 粘贴快加：非输入框 paste URL → 查重 → 自动抓取标题描述图标
 */
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import MechaShell from '@/app/layouts/MechaShell.vue'
import SiteCard from '@/modules/portal/components/SiteCard.vue'
import ContextMenu from '@/modules/portal/components/ContextMenu.vue'
import WebsiteFormModal from '@/modules/portal/components/WebsiteFormModal.vue'
import DuplicateDialog from '@/modules/portal/components/DuplicateDialog.vue'
import AnnouncementModal from '@/modules/portal/components/AnnouncementModal.vue'
import AppIcon from '@/shared/ui/AppIcon.vue'
import { iconForCategory } from '@/shared/icons/registry'
import { useAuthStore } from '@/shared/stores/auth'
import { usePortalStore } from '@/shared/stores/portal'
import { useToast } from '@/shared/composables/useToast'
import type { Category, Website } from '@/shared/types/models'

const auth = useAuthStore()
const portal = usePortalStore()
const toast = useToast()
const router = useRouter()

/** 当前舱段激活的 Tab：root | 子分类 id */
const activeTabs = ref<Record<number, number | 'root'>>({})
/** 已展开全部的 key：`${catId}:${tab}` */
const expandedKeys = ref<Record<string, boolean>>({})
/** 分类/子类全量缓存 */
const sectionSitesCache = ref<Record<number, Website[]>>({})

const ctx = ref<{ x: number; y: number; site: Website } | null>(null)
const formOpen = ref(false)
const formMode = ref<'create' | 'edit'>('create')
const formSite = ref<Website | null>(null)
const formUrl = ref('')
const formLoading = ref(false)
const formProgress = ref('')
const formRef = ref<InstanceType<typeof WebsiteFormModal> | null>(null)
const dupOpen = ref(false)
const dupSite = ref<Website | null>(null)
const pendingForceUrl = ref('')
const dragState = ref<{ catId: number; ids: number[] } | null>(null)
const showTop = ref(false)
const useAI = ref(false)

const displayCategories = computed(() => portal.categories)

const aiAvailable = computed(() => {
  const s = portal.settings
  if (!s?.ai_search_enabled) return false
  if (!auth.isLoggedIn && !s.ai_search_allow_anonymous) return false
  return true
})

function tabKey(catId: number, tab: number | 'root' | undefined) {
  return `${catId}:${tab ?? 'root'}`
}

function resolveTab(cat: Category): number | 'root' {
  return activeTabs.value[cat.id] ?? (cat.displayed_subcategory_id ? cat.displayed_subcategory_id : 'root')
}

function cacheIdFor(cat: Category): number {
  const tab = resolveTab(cat)
  return tab === 'root' ? cat.id : tab
}

/** 当前 Tab 下链接总数（用于展开按钮） */
function totalFor(cat: Category): number {
  const tab = resolveTab(cat)
  if (typeof tab === 'number') {
    const ch = cat.children?.find((c) => c.id === tab)
    if (ch?.website_count != null) return ch.website_count
    return sectionSitesCache.value[tab]?.length ?? 0
  }
  // root / 未分类
  if (cat.direct_count != null) return cat.direct_count
  return sectionSitesCache.value[cat.id]?.length ?? cat.websites?.length ?? 0
}

function isExpanded(cat: Category) {
  return !!expandedKeys.value[tabKey(cat.id, resolveTab(cat))]
}

function limitOf(cat: Category) {
  return cat.display_limit > 0 ? cat.display_limit : 10
}

function sitesFor(cat: Category): Website[] {
  const cid = cacheIdFor(cat)
  const all = sectionSitesCache.value[cid] || cat.websites || []
  if (isExpanded(cat)) return all
  const lim = limitOf(cat)
  // 首页首屏：未展开时始终限显
  return all.slice(0, lim)
}

function canExpand(cat: Category) {
  return totalFor(cat) > sitesFor(cat).length || (totalFor(cat) > limitOf(cat) && !isExpanded(cat))
}

async function ensureFullList(cat: Category) {
  const cid = cacheIdFor(cat)
  // 若缓存长度仍像限显且总数更大，强制拉全量
  const total = totalFor(cat)
  const cached = sectionSitesCache.value[cid]
  if (cached && cached.length >= total && total > 0) return cached
  const list = await portal.loadCategoryAll(cid)
  sectionSitesCache.value[cid] = list
  // 修正总数展示：用真实长度
  return list
}

async function expandCategory(cat: Category) {
  formProgress.value = ''
  try {
    await ensureFullList(cat)
    expandedKeys.value = {
      ...expandedKeys.value,
      [tabKey(cat.id, resolveTab(cat))]: true,
    }
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '加载失败')
  }
}

function collapseCategory(cat: Category) {
  expandedKeys.value = {
    ...expandedKeys.value,
    [tabKey(cat.id, resolveTab(cat))]: false,
  }
}

async function selectTab(cat: Category, tab: number | 'root') {
  activeTabs.value[cat.id] = tab
  // 切换 Tab 时默认收起，保持轻量
  const key = tabKey(cat.id, tab)
  if (!expandedKeys.value[key]) {
    const cid = tab === 'root' ? cat.id : tab
    // 先用已有缓存；没有则拉全量但仍然限显（slice）
    if (!sectionSitesCache.value[cid]) {
      try {
        sectionSitesCache.value[cid] = await portal.loadCategoryAll(cid)
      } catch {
        sectionSitesCache.value[cid] = []
      }
    }
  }
}

async function onSidebarSubcategory(parentId: number, childId: number | 'root') {
  const cat = portal.categories.find((c) => c.id === parentId)
  if (!cat) return
  await selectTab(cat, childId)
}

function initTabs() {
  for (const cat of portal.categories) {
    if (cat.displayed_subcategory_id) {
      activeTabs.value[cat.id] = cat.displayed_subcategory_id
      // 首页只带了限显列表
      sectionSitesCache.value[cat.displayed_subcategory_id] = cat.websites || []
    } else {
      activeTabs.value[cat.id] = 'root'
      sectionSitesCache.value[cat.id] = cat.websites || []
    }
  }
}

watch(
  () => portal.categories,
  () => initTabs(),
  { deep: true },
)

function onScroll() {
  showTop.value = window.scrollY > 400
}

onMounted(async () => {
  await portal.loadHome()
  initTabs()
  if (location.hash.startsWith('#cat-')) {
    document.querySelector(location.hash)?.scrollIntoView()
  }
  window.addEventListener('paste', onPaste)
  window.addEventListener('scroll', onScroll, { passive: true })
})

onUnmounted(() => {
  window.removeEventListener('paste', onPaste)
  window.removeEventListener('scroll', onScroll)
})

function isValidUrl(text: string) {
  try {
    const u = new URL(text.includes('://') ? text : `https://${text}`)
    return u.protocol === 'http:' || u.protocol === 'https:'
  } catch {
    return false
  }
}

async function onPaste(e: ClipboardEvent) {
  if (!auth.isAdmin) return
  const t = e.target as HTMLElement | null
  if (t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.isContentEditable)) return
  if (formOpen.value || dupOpen.value) return
  const text = e.clipboardData?.getData('text')?.trim() || ''
  if (!isValidUrl(text)) return
  e.preventDefault()
  await startQuickAdd(text)
}

async function startQuickAdd(url: string) {
  formMode.value = 'create'
  formSite.value = null
  formUrl.value = url
  formOpen.value = true
  formLoading.value = true
  formProgress.value = '正在检查是否重复…'
  await nextTick()
  formRef.value?.setUrl(url)
  try {
    const check = await portal.checkUrl(url)
    if (check.exists && check.website) {
      formOpen.value = false
      formLoading.value = false
      formProgress.value = ''
      dupSite.value = check.website
      pendingForceUrl.value = url
      dupOpen.value = true
      return
    }
    formProgress.value = '正在抓取标题、描述与图标…'
    const meta = await portal.fetchSite(url)
    await nextTick()
    formRef.value?.applyMeta({
      title: String(meta.title || ''),
      description: String(meta.description || ''),
      icon_url: String(meta.icon_url || ''),
      url: String(meta.url || url),
    })
    if (!meta.title && !meta.description) {
      toast.info('未能自动解析，请手动填写')
    }
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : '解析失败')
    formRef.value?.setUrl(url)
  } finally {
    formLoading.value = false
    formProgress.value = ''
  }
}

async function onDupForce() {
  dupOpen.value = false
  await startQuickAdd(pendingForceUrl.value)
  // startQuickAdd 会再 check 一次；force 时跳过重复：直接抓取
}

// 强制添加：跳过查重
async function onDupForceSkipCheck() {
  dupOpen.value = false
  const url = pendingForceUrl.value
  formMode.value = 'create'
  formSite.value = null
  formUrl.value = url
  formOpen.value = true
  formLoading.value = true
  formProgress.value = '正在抓取网站信息…'
  await nextTick()
  formRef.value?.setUrl(url)
  try {
    const meta = await portal.fetchSite(url)
    await nextTick()
    formRef.value?.applyMeta({
      title: String(meta.title || ''),
      description: String(meta.description || ''),
      icon_url: String(meta.icon_url || ''),
      url,
    })
  } finally {
    formLoading.value = false
    formProgress.value = ''
  }
}

function onDupView() {
  dupOpen.value = false
  const w = dupSite.value
  if (!w) return
  // 定位分类：优先 category_id
  if (w.category_id) {
    // 可能是子分类：找到父级
    let parentId = w.category_id
    let childTab: number | 'root' = 'root'
    for (const cat of portal.categories) {
      if (cat.id === w.category_id) {
        parentId = cat.id
        childTab = 'root'
        break
      }
      const ch = cat.children?.find((c) => c.id === w.category_id)
      if (ch) {
        parentId = cat.id
        childTab = ch.id
        break
      }
    }
    const parent = portal.categories.find((c) => c.id === parentId)
    if (parent) {
      selectTab(parent, childTab).then(() => {
        expandCategory(parent).then(() => {
          document.getElementById(`cat-${parentId}`)?.scrollIntoView({ behavior: 'smooth' })
          // 高亮卡片
          nextTick(() => {
            const card = document.querySelector(`[data-id="${w.id}"]`) as HTMLElement | null
            card?.classList.add('site-card--highlight')
            setTimeout(() => card?.classList.remove('site-card--highlight'), 2000)
          })
        })
      })
    }
  }
  toast.info('已定位到已有链接')
}

async function onFormSubmit(payload: Record<string, unknown>) {
  formLoading.value = true
  formProgress.value = '正在保存…'
  try {
    if (formMode.value === 'create') {
      await portal.createWebsite(payload)
      toast.success('链接已添加')
    } else if (formSite.value) {
      await portal.updateWebsite(formSite.value.id, payload)
      toast.success('已保存')
    }
    formOpen.value = false
    // 刷新后重新 initTabs
    initTabs()
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : '操作失败')
  } finally {
    formLoading.value = false
    formProgress.value = ''
  }
}

async function onFetchMeta(url: string) {
  formLoading.value = true
  formProgress.value = '正在抓取网站信息…'
  try {
    const meta = await portal.fetchSite(url)
    formRef.value?.applyMeta({
      title: String(meta.title || ''),
      description: String(meta.description || ''),
      icon_url: String(meta.icon_url || ''),
      url: String(meta.url || url),
    })
    toast.success('已填充网站信息')
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : '抓取失败')
  } finally {
    formLoading.value = false
    formProgress.value = ''
  }
}

/**
 * 对齐旧站 index.html 卡片：
 *   <a href="/goto/id 或 直达" target="_blank">
 * - 始终新标签打开
 * - 开过渡：新标签 → /goto/:id（访问计数在 GotoPage 的 visit）
 * - 不开过渡 / 用户选过不再显示：新标签直接目标 URL
 * 过渡页内：location.replace 同标签离开（见 GotoPage）
 */
async function openSite(site: Website) {
  let skip = false
  try {
    skip = localStorage.getItem('booknav_skip_transition') === '1'
  } catch {
    /* ignore */
  }

  const settings = portal.settings
  const enableTransition = !!settings?.enable_transition
  // 访客倒计时；管理员倒计时由后端 visit 再算，前端用公共 settings 近似
  const countdown = settings?.transition_time ?? 0

  if (enableTransition && countdown > 0 && !skip) {
    // 仅开过渡页；计数留给 /goto 内 visit，避免双计
    window.open(`${location.origin}/goto/${site.id}`, '_blank', 'noopener,noreferrer')
    return
  }

  try {
    const data = await portal.visit(site.id)
    window.open(data.website?.url || site.url, '_blank', 'noopener,noreferrer')
  } catch {
    window.open(site.url, '_blank', 'noopener,noreferrer')
  }
}

function onContext(e: MouseEvent, site: Website) {
  ctx.value = { x: e.clientX, y: e.clientY, site }
}

async function ctxVisit() {
  if (!ctx.value) return
  await openSite(ctx.value.site)
  ctx.value = null
}

async function ctxCopy() {
  if (!ctx.value) return
  try {
    await navigator.clipboard.writeText(ctx.value.site.url)
    toast.success('已复制链接')
  } catch {
    toast.error('复制失败')
  }
  ctx.value = null
}

function ctxEdit() {
  if (!ctx.value) return
  formMode.value = 'edit'
  formSite.value = ctx.value.site
  formUrl.value = ctx.value.site.url
  formOpen.value = true
  ctx.value = null
}

function ctxAdd() {
  ctx.value = null
  formMode.value = 'create'
  formSite.value = null
  formUrl.value = ''
  formOpen.value = true
}

async function ctxRemove() {
  if (!ctx.value) return
  const site = ctx.value.site
  ctx.value = null
  if (!confirm(`删除「${site.title}」？`)) return
  try {
    await portal.deleteWebsite(site.id)
    toast.success('已删除')
    initTabs()
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : '删除失败')
  }
}

function onDragStart(cat: Category, site: Website, e: DragEvent) {
  if (!auth.isAdmin) return
  e.dataTransfer?.setData('text/site-id', String(site.id))
  e.dataTransfer!.effectAllowed = 'move'
  dragState.value = { catId: cat.id, ids: sitesFor(cat).map((s) => s.id) }
}

function onDrop(cat: Category, target: Website, e: DragEvent) {
  e.preventDefault()
  if (!auth.isAdmin || !dragState.value) return
  const fromId = Number(e.dataTransfer?.getData('text/site-id'))
  const ids = [...dragState.value.ids]
  const from = ids.indexOf(fromId)
  const to = ids.indexOf(target.id)
  if (from < 0 || to < 0 || from === to) return
  ids.splice(from, 1)
  ids.splice(to, 0, fromId)
  const cid = cacheIdFor(cat)
  const map = new Map((sectionSitesCache.value[cid] || sitesFor(cat)).map((s) => [s.id, s]))
  // 合并：只重排当前可见顺序到缓存前部
  const rest = (sectionSitesCache.value[cid] || []).filter((s) => !ids.includes(s.id))
  sectionSitesCache.value[cid] = [...ids.map((id) => map.get(id)!).filter(Boolean), ...rest]
  portal
    .reorderWebsites(cid, sectionSitesCache.value[cid].map((s) => s.id))
    .then(() => toast.success('排序已保存'))
    .catch((err) => toast.error(err.message || '排序失败'))
}

async function onSearch(q: string) {
  await portal.search(q, useAI.value && aiAvailable.value)
}

function scrollTop() {
  window.scrollTo({ top: 0, behavior: 'smooth' })
}
</script>

<template>
  <MechaShell
    :categories="displayCategories"
    :ai-available="aiAvailable"
    v-model:use-ai="useAI"
    @search="onSearch"
    @select-subcategory="onSidebarSubcategory"
  >
    <div v-if="portal.error && !portal.categories.length" class="state error">{{ portal.error }}</div>
    <!-- skeleton while first home payload arrives — keep shell visible, avoid blank flash -->
    <div v-else-if="portal.loading && !portal.categories.length" class="home-skeleton" aria-busy="true">
      <div v-for="n in 3" :key="n" class="home-skeleton__bay">
        <div class="home-skeleton__title" />
        <div class="home-skeleton__grid">
          <div v-for="m in 8" :key="m" class="home-skeleton__card" />
        </div>
      </div>
    </div>

    <template v-else-if="!portal.loading || portal.categories.length">
      <!-- 搜索结果 -->
      <section v-if="portal.searchResults" class="search-layer">
        <header class="bay-header">
          <span class="bay-header__icon bay-header__icon--magenta">
            <AppIcon name="search" :size="18" />
          </span>
          <h2 class="bay-header__title">搜索结果</h2>
          <span class="bay-header__meta">
            {{ portal.searchQuery }} · {{ portal.searchResults.length }} 条
            <template v-if="useAI && aiAvailable"> · AI</template>
          </span>
          <button type="button" class="m-btn m-btn--ghost" @click="portal.clearSearch()">返回导航</button>
        </header>
        <div v-if="!portal.searchResults.length" class="state">没有匹配的链接</div>
        <div v-else class="card-grid">
          <SiteCard
            v-for="site in portal.searchResults"
            :key="site.id"
            :site="site"
            @open="openSite"
            @context="onContext"
          />
        </div>
      </section>

      <template v-else>
        <!-- 精选 -->
        <section v-if="portal.featured.length" class="featured">
          <header class="bay-header">
            <span class="bay-header__icon bay-header__icon--amber">
              <AppIcon name="star" :size="18" />
            </span>
            <h2 class="bay-header__title">精选</h2>
            <span class="bay-header__meta">{{ portal.featured.length }}</span>
          </header>
          <div class="card-grid">
            <SiteCard
              v-for="site in portal.featured"
              :key="'f' + site.id"
              :site="site"
              @open="openSite"
              @context="onContext"
            />
          </div>
        </section>

        <!-- 分类舱段 -->
        <section
          v-for="cat in displayCategories"
          :id="`cat-${cat.id}`"
          :key="cat.id"
          class="category-section"
        >
          <header class="bay-header">
            <span class="bay-header__icon">
              <AppIcon :name="iconForCategory(cat.icon, cat.id)" :size="18" />
            </span>
            <h2 class="bay-header__title">{{ cat.name }}</h2>
            <span class="bay-header__meta">
              {{ sitesFor(cat).length }} / {{ totalFor(cat) }}
            </span>
            <button
              v-if="totalFor(cat) > limitOf(cat)"
              type="button"
              class="m-btn m-btn--ghost expand-btn"
              @click="isExpanded(cat) ? collapseCategory(cat) : expandCategory(cat)"
            >
              {{ isExpanded(cat) ? '收起' : '展开全部' }}
            </button>
          </header>

          <div v-if="cat.children?.length" class="m-tabs" role="tablist">
            <button
              type="button"
              class="m-tab"
              :class="{ 'm-tab--active': resolveTab(cat) === 'root' }"
              @click="selectTab(cat, 'root')"
            >
              未分类 ({{ cat.direct_count ?? 0 }})
            </button>
            <button
              v-for="ch in cat.children"
              :key="ch.id"
              type="button"
              class="m-tab"
              :class="{ 'm-tab--active': resolveTab(cat) === ch.id }"
              @click="selectTab(cat, ch.id)"
            >
              {{ ch.name }} ({{ ch.website_count ?? 0 }})
            </button>
          </div>

          <div class="card-grid">
            <SiteCard
              v-for="site in sitesFor(cat)"
              :key="site.id"
              :site="site"
              :draggable="auth.isAdmin"
              @open="openSite"
              @context="onContext"
              @dragstart="(e: DragEvent) => onDragStart(cat, site, e)"
              @drop="(e: DragEvent) => onDrop(cat, site, e)"
            />
          </div>

          <div v-if="!sitesFor(cat).length" class="empty-hint">该分类下暂无链接</div>
        </section>

        <p v-if="!displayCategories.length" class="state">暂无分类</p>
      </template>
    </template>

    <ContextMenu
      v-if="ctx"
      :x="ctx.x"
      :y="ctx.y"
      :is-admin="auth.isAdmin"
      @close="ctx = null"
      @visit="ctxVisit"
      @copy="ctxCopy"
      @edit="ctxEdit"
      @add="ctxAdd"
      @remove="ctxRemove"
    />

    <WebsiteFormModal
      ref="formRef"
      :open="formOpen"
      :mode="formMode"
      :site="formSite"
      :categories="displayCategories"
      :initial-url="formUrl"
      :loading="formLoading"
      :progress-text="formProgress"
      @close="formOpen = false"
      @submit="onFormSubmit"
      @fetch-meta="onFetchMeta"
    />

    <DuplicateDialog
      :open="dupOpen"
      :website="dupSite"
      @cancel="dupOpen = false"
      @view="onDupView"
      @force="onDupForceSkipCheck"
    />

    <!-- 公告弹窗（非内嵌条） -->
    <AnnouncementModal
      v-if="portal.settings"
      :enabled="!!portal.settings.announcement_enabled"
      :title="portal.settings.announcement_title"
      :content="portal.settings.announcement_content"
      :remember-days="portal.settings.announcement_remember_days || 7"
    />

    <button
      v-show="showTop"
      type="button"
      class="back-top m-btn m-btn--primary"
      aria-label="回到顶部"
      @click="scrollTop"
    >
      <AppIcon name="chevron-down" :size="18" class="back-top__icon" />
    </button>
  </MechaShell>
</template>

<style scoped>
.state {
  padding: 48px;
  text-align: center;
  color: var(--text-secondary);
}
.state.error {
  color: var(--danger);
}
.featured {
  margin-bottom: 36px;
}
.category-section {
  scroll-margin-top: calc(var(--top-rail-height) + 16px);
  margin-bottom: 44px;
}
.card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  /* fixed row rhythm — cards must not stretch grid by art */
  grid-auto-rows: 96px;
  align-items: stretch;
  gap: 14px;
}
.empty-hint {
  color: var(--text-muted);
  font-size: var(--text-sm);
  padding: 12px 0;
}
.expand-btn {
  height: 30px !important;
  font-size: 12px !important;
  flex-shrink: 0;
  letter-spacing: 0.04em;
  border-color: color-mix(in srgb, var(--energy) 35%, var(--stroke-dim)) !important;
  color: var(--energy) !important;
}
.back-top {
  position: fixed;
  right: 20px;
  bottom: 24px;
  z-index: 40;
  width: 40px !important;
  height: 40px !important;
  padding: 0 !important;
  border-radius: 4px;
  box-shadow: var(--glow-sm);
}
.back-top__icon {
  transform: rotate(180deg);
}
:deep(.site-card--highlight) {
  border-color: var(--energy) !important;
  box-shadow: var(--glow-md) !important;
}
</style>
