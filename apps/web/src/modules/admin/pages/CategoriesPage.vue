<script setup lang="ts">
/**
 * Category admin — collapsible tree (roots collapsed by default).
 * Sort: sibling drag + pin top/bottom + adjacent ↑↓.
 */
import { computed, onMounted, ref, watch } from 'vue'
import { apiGet, apiPost, apiDelete, apiPatch, apiPut } from '@/shared/api/client'
import { useToast } from '@/shared/composables/useToast'
import type { Category } from '@/shared/types/models'
import CategoryIconPicker from '../components/CategoryIconPicker.vue'
import AppIcon from '@/shared/ui/AppIcon.vue'
import { iconForCategory } from '@/shared/icons/registry'

const EXPAND_KEY = 'booknav_admin_cat_expanded'

const toast = useToast()
const categories = ref<Category[]>([])
const loading = ref(false)
const sorting = ref(false)
const query = ref('')
const expanded = ref<Set<number>>(new Set())
const dragFrom = ref<{ id: number; parentId: number | null } | null>(null)
const dragOverId = ref<number | null>(null)
/** Block drag when starting from buttons / inputs */
let blockDrag = false
/** After a drag, ignore the synthetic click that would toggle expand */
let suppressClick = false

/** Inline edit form (bound when editingId set). */
const form = ref({
  name: '',
  icon: 'folder',
  description: '',
  display_limit: 10,
  parent_id: null as number | null,
})
const editingId = ref<number | null>(null)

function loadExpandState() {
  try {
    const raw = sessionStorage.getItem(EXPAND_KEY)
    if (!raw) return
    const ids = JSON.parse(raw) as number[]
    if (Array.isArray(ids)) expanded.value = new Set(ids.filter((n) => typeof n === 'number'))
  } catch {
    /* ignore */
  }
}

function persistExpand() {
  sessionStorage.setItem(EXPAND_KEY, JSON.stringify([...expanded.value]))
}

function isOpen(id: number) {
  return expanded.value.has(id)
}

function toggle(id: number) {
  const next = new Set(expanded.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expanded.value = next
  persistExpand()
}

function expandAll() {
  const next = new Set<number>()
  for (const c of categories.value) {
    if (c.children?.length) next.add(c.id)
  }
  expanded.value = next
  persistExpand()
}

function collapseAll() {
  expanded.value = new Set()
  persistExpand()
}

function childCount(c: Category) {
  return c.children?.length ?? 0
}

function linkCount(c: Category) {
  return c.website_count ?? c.direct_count ?? 0
}

/** Parent options for form (roots only labels + depth for nested if ever needed). */
const parentOptions = computed(() => {
  const out: { id: number; label: string }[] = []
  for (const c of categories.value) {
    out.push({ id: c.id, label: c.name })
    for (const ch of c.children || []) {
      out.push({ id: ch.id, label: `— ${ch.name}` })
    }
  }
  return out
})

const filteredRoots = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return categories.value
  return categories.value
    .map((root) => {
      const nameHit = root.name.toLowerCase().includes(q)
      const kids = (root.children || []).filter((ch) => ch.name.toLowerCase().includes(q))
      if (nameHit) return root
      if (kids.length) return { ...root, children: kids }
      return null
    })
    .filter(Boolean) as Category[]
})

// Auto-expand parents when search matches a child
watch(query, (q) => {
  if (!q.trim()) return
  const next = new Set(expanded.value)
  for (const root of categories.value) {
    const kids = root.children || []
    if (kids.some((ch) => ch.name.toLowerCase().includes(q.trim().toLowerCase()))) {
      next.add(root.id)
    }
  }
  expanded.value = next
})

function parentIdOf(c: Category): number | null {
  const p = c.parent_id
  if (p == null || p === 0) return null
  return p
}

function siblingList(c: Category): Category[] {
  const pid = parentIdOf(c)
  if (pid == null) return categories.value
  const parent = findCat(categories.value, pid)
  return parent?.children || []
}

function findCat(list: Category[], id: number): Category | null {
  for (const c of list) {
    if (c.id === id) return c
    if (c.children?.length) {
      const hit = findCat(c.children, id)
      if (hit) return hit
    }
  }
  return null
}

/** Apply new sibling order in local state (optimistic). */
function reorderLocal(parentId: number | null, ids: number[]) {
  if (parentId == null) {
    const map = new Map(categories.value.map((c) => [c.id, c]))
    categories.value = ids.map((id) => map.get(id)!).filter(Boolean)
    return
  }
  categories.value = categories.value.map((root) => {
    if (root.id !== parentId) return root
    const map = new Map((root.children || []).map((c) => [c.id, c]))
    return { ...root, children: ids.map((id) => map.get(id)!).filter(Boolean) }
  })
}

async function load() {
  loading.value = true
  try {
    categories.value = await apiGet('/api/v1/admin/categories')
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

async function applyOrder(parentId: number | null, ids: number[]) {
  if (ids.length < 2) return
  const prev = categories.value
  reorderLocal(parentId, ids)
  sorting.value = true
  try {
    await apiPut('/api/v1/portal/categories/order', { ids })
    toast.success('排序已保存')
  } catch (e: unknown) {
    categories.value = prev
    toast.error(e instanceof Error ? e.message : '排序失败')
    await load()
  } finally {
    sorting.value = false
  }
}

async function moveSibling(c: Category, dir: -1 | 1) {
  const sibs = siblingList(c)
  const idx = sibs.findIndex((x) => x.id === c.id)
  const j = idx + dir
  if (idx < 0 || j < 0 || j >= sibs.length) return
  const ids = sibs.map((x) => x.id)
  ;[ids[idx], ids[j]] = [ids[j], ids[idx]]
  await applyOrder(parentIdOf(c), ids)
}

async function movePin(c: Category, where: 'top' | 'bottom') {
  const sibs = siblingList(c)
  const ids = sibs.map((x) => x.id).filter((id) => id !== c.id)
  if (where === 'top') ids.unshift(c.id)
  else ids.push(c.id)
  await applyOrder(parentIdOf(c), ids)
}

function armNoDrag() {
  blockDrag = true
}

function onDragStart(c: Category, e: DragEvent) {
  if (sorting.value || blockDrag) {
    e.preventDefault()
    blockDrag = false
    return
  }
  const t = e.target as HTMLElement | null
  // twist button is inside main — do not start drag from it
  if (t?.closest?.('button, a, input, select, textarea')) {
    e.preventDefault()
    return
  }
  const pid = parentIdOf(c)
  dragFrom.value = { id: c.id, parentId: pid }
  try {
    e.dataTransfer!.effectAllowed = 'move'
    e.dataTransfer!.setData('text/plain', String(c.id))
    e.dataTransfer!.setData('text/cat-id', String(c.id))
  } catch {
    /* ignore */
  }
}

function onDragOver(c: Category, e: DragEvent) {
  if (!dragFrom.value) return
  if (dragFrom.value.parentId !== parentIdOf(c)) return
  if (dragFrom.value.id === c.id) return
  e.preventDefault()
  e.stopPropagation()
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
  dragOverId.value = c.id
}

function onDragLeave(c: Category) {
  if (dragOverId.value === c.id) dragOverId.value = null
}

async function onDrop(c: Category, e: DragEvent) {
  e.preventDefault()
  e.stopPropagation()
  const raw = e.dataTransfer?.getData('text/cat-id') || e.dataTransfer?.getData('text/plain')
  const fromId = Number(raw || dragFrom.value?.id)
  const fromParent = dragFrom.value?.parentId ?? null
  dragOverId.value = null
  dragFrom.value = null
  if (!fromId || fromId === c.id) return
  const toParent = parentIdOf(c)
  if (fromParent !== toParent) {
    toast.error('只能在同一级内排序')
    return
  }
  const sibs = siblingList(c)
  const ids = sibs.map((x) => x.id)
  const from = ids.indexOf(fromId)
  const to = ids.indexOf(c.id)
  if (from < 0 || to < 0 || from === to) return
  suppressClick = true
  ids.splice(from, 1)
  ids.splice(to, 0, fromId)
  await applyOrder(toParent, ids)
}

function onDragEnd() {
  if (dragFrom.value) suppressClick = true
  dragFrom.value = null
  dragOverId.value = null
  blockDrag = false
  window.setTimeout(() => {
    suppressClick = false
  }, 80)
}

/** Click row body (not buttons) to expand/collapse parents with children. */
function onNodeActivate(c: Category, e: MouseEvent) {
  if (suppressClick) {
    suppressClick = false
    return
  }
  const t = e.target as HTMLElement | null
  if (t?.closest?.('button, a, input, select, textarea, .cat-node__actions, .cat-node__sort')) {
    return
  }
  // parent with children: click icon / title / empty area toggles
  if (childCount(c) > 0 && parentIdOf(c) == null) {
    toggle(c.id)
  }
}

const createForm = ref({
  name: '',
  icon: 'folder',
  description: '',
  display_limit: 10,
  parent_id: null as number | null,
})

function startEdit(c: Category) {
  // toggle off if same row
  if (editingId.value === c.id) {
    resetEdit()
    return
  }
  editingId.value = c.id
  form.value = {
    name: c.name,
    icon: iconForCategory(c.icon, c.id),
    description: c.description || '',
    display_limit: c.display_limit || 10,
    parent_id: c.parent_id ?? null,
  }
  // ensure parent expanded when editing a child
  const pid = parentIdOf(c)
  if (pid != null && !expanded.value.has(pid)) {
    const next = new Set(expanded.value)
    next.add(pid)
    expanded.value = next
    persistExpand()
  }
}

function resetEdit() {
  editingId.value = null
  form.value = {
    name: '',
    icon: 'folder',
    description: '',
    display_limit: 10,
    parent_id: null,
  }
}

function resetCreate() {
  createForm.value = {
    name: '',
    icon: 'folder',
    description: '',
    display_limit: 10,
    parent_id: null,
  }
}

async function saveEdit() {
  if (!editingId.value) return
  if (!form.value.name.trim()) {
    toast.error('请填写名称')
    return
  }
  try {
    await apiPatch(`/api/v1/admin/categories/${editingId.value}`, {
      name: form.value.name.trim(),
      icon: form.value.icon || 'folder',
      description: form.value.description,
      display_limit: form.value.display_limit,
      parent_id: form.value.parent_id,
    })
    toast.success('已更新')
    resetEdit()
    await load()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '失败')
  }
}

async function saveCreate() {
  if (!createForm.value.name.trim()) {
    toast.error('请填写名称')
    return
  }
  try {
    await apiPost('/api/v1/admin/categories', {
      name: createForm.value.name.trim(),
      icon: createForm.value.icon || 'folder',
      description: createForm.value.description,
      display_limit: createForm.value.display_limit,
      parent_id: createForm.value.parent_id,
    })
    toast.success('已创建')
    resetCreate()
    await load()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '失败')
  }
}

async function remove(id: number) {
  if (!confirm('删除分类？（有子分类或链接时会失败）')) return
  try {
    await apiDelete(`/api/v1/admin/categories/${id}`)
    toast.success('已删除')
    if (editingId.value === id) resetEdit()
    await load()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '删除失败')
  }
}

function sibIndex(c: Category) {
  return siblingList(c).findIndex((x) => x.id === c.id)
}

function sibCount(c: Category) {
  return siblingList(c).length
}

onMounted(() => {
  loadExpandState()
  load()
})
</script>

<template>
  <div class="admin-page cat-page">
    <header class="page-header">
      <h1>分类管理</h1>
      <div class="page-header__actions">
        <button type="button" class="c-btn c-btn--ghost c-btn--sm" :disabled="loading" @click="expandAll">
          全部展开
        </button>
        <button type="button" class="c-btn c-btn--ghost c-btn--sm" :disabled="loading" @click="collapseAll">
          全部折叠
        </button>
        <button type="button" class="c-btn c-btn--ghost c-btn--sm" :disabled="loading" @click="load">刷新</button>
      </div>
    </header>

    <div class="c-card c-card__body form-card">
      <h3 class="c-card__title">新建分类</h3>
      <div class="form-row">
        <input v-model="createForm.name" class="c-input" placeholder="名称" style="max-width: 200px" />
        <input
          v-model.number="createForm.display_limit"
          class="c-input"
          type="number"
          min="1"
          style="max-width: 100px"
          title="首页展示条数"
        />
        <select v-model="createForm.parent_id" class="c-input" style="max-width: 200px">
          <option :value="null">一级分类</option>
          <option v-for="p in parentOptions" :key="p.id" :value="p.id">{{ p.label }}</option>
        </select>
        <button type="button" class="c-btn c-btn--primary" @click="saveCreate">创建</button>
      </div>
      <div class="form-icon">
        <span class="form-icon__label">图标</span>
        <CategoryIconPicker v-model="createForm.icon" />
      </div>
    </div>

    <div class="tree-toolbar">
      <div class="tree-search">
        <AppIcon name="search" :size="16" />
        <input v-model="query" class="c-input" type="search" placeholder="筛选分类名称…" />
      </div>
      <span class="tree-meta">
        {{ categories.length }} 个一级
        <template v-if="sorting"> · 保存中…</template>
      </span>
    </div>

    <div v-if="loading" class="c-empty">加载中…</div>
    <div v-else-if="!filteredRoots.length" class="c-empty">
      {{ query.trim() ? '无匹配分类' : '暂无分类' }}
    </div>

    <div v-else class="cat-tree" :class="{ 'cat-tree--busy': sorting }">
      <div
        v-for="root in filteredRoots"
        :key="root.id"
        class="cat-branch"
        :class="{ 'cat-branch--open': isOpen(root.id) && childCount(root) > 0 }"
      >
        <!-- Parent: drag zone = main (no nested buttons); click main to expand -->
        <div
          class="cat-node cat-node--root"
          :class="{
            'cat-node--drag-over': dragOverId === root.id,
            'cat-node--dragging': dragFrom?.id === root.id,
            'cat-node--editing': editingId === root.id,
            'cat-node--expandable': childCount(root) > 0,
          }"
          @dragover="onDragOver(root, $event)"
          @dragleave="onDragLeave(root)"
          @drop="onDrop(root, $event)"
        >
          <div class="cat-node__rail" aria-hidden="true" />

          <div
            class="cat-node__main"
            :draggable="!sorting"
            @dragstart="onDragStart(root, $event)"
            @dragend="onDragEnd"
            @click="onNodeActivate(root, $event)"
          >
            <button
              v-if="childCount(root) > 0"
              type="button"
              class="cat-twist"
              :class="{ 'cat-twist--open': isOpen(root.id) }"
              :aria-expanded="isOpen(root.id)"
              :title="isOpen(root.id) ? '折叠' : '展开'"
              @mousedown.stop="armNoDrag"
              @click.stop="toggle(root.id)"
            >
              <span class="cat-twist__glyph" aria-hidden="true" />
            </button>
            <span v-else class="cat-twist cat-twist--leaf" aria-hidden="true" />

            <span class="cat-node__grip" title="拖拽排序" aria-hidden="true">
              <AppIcon name="grip-vertical" :size="14" />
            </span>

            <span class="cat-node__icon">
              <AppIcon :name="iconForCategory(root.icon, root.id)" :size="18" />
            </span>

            <div class="cat-node__body">
              <div class="cat-node__title">{{ root.name }}</div>
              <div class="cat-node__chips">
                <span v-if="childCount(root)" class="chip chip--sub">{{ childCount(root) }} 子类</span>
                <span class="chip">{{ linkCount(root) }} 链接</span>
                <span class="chip chip--mute">展示 {{ root.display_limit }}</span>
              </div>
            </div>
          </div>

          <div class="cat-node__sort" @mousedown="armNoDrag" @click.stop>
            <button
              type="button"
              class="sort-mini"
              title="置顶"
              :disabled="sorting || sibIndex(root) <= 0"
              @click="movePin(root, 'top')"
            >
              ⤒
            </button>
            <button
              type="button"
              class="sort-mini"
              title="上移"
              :disabled="sorting || sibIndex(root) <= 0"
              @click="moveSibling(root, -1)"
            >
              ↑
            </button>
            <button
              type="button"
              class="sort-mini"
              title="下移"
              :disabled="sorting || sibIndex(root) >= sibCount(root) - 1"
              @click="moveSibling(root, 1)"
            >
              ↓
            </button>
            <button
              type="button"
              class="sort-mini"
              title="置底"
              :disabled="sorting || sibIndex(root) >= sibCount(root) - 1"
              @click="movePin(root, 'bottom')"
            >
              ⤓
            </button>
          </div>

          <div class="cat-node__actions" @mousedown="armNoDrag" @click.stop>
            <button
              type="button"
              class="c-btn c-btn--ghost c-btn--sm"
              :class="{ 'c-btn--active-edit': editingId === root.id }"
              @click="startEdit(root)"
            >
              {{ editingId === root.id ? '收起' : '编辑' }}
            </button>
            <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="remove(root.id)">删除</button>
          </div>
        </div>

        <!-- Inline edit under this root -->
        <div v-if="editingId === root.id" class="cat-inline-edit" @mousedown="armNoDrag" @click.stop>
          <div class="cat-inline-edit__head">
            <span class="cat-inline-edit__badge">编辑</span>
            <span class="cat-inline-edit__name">{{ root.name }}</span>
          </div>
          <div class="form-row">
            <input v-model="form.name" class="c-input" placeholder="名称" style="max-width: 200px" />
            <input
              v-model.number="form.display_limit"
              class="c-input"
              type="number"
              min="1"
              style="max-width: 100px"
              title="首页展示条数"
            />
            <select v-model="form.parent_id" class="c-input" style="max-width: 200px">
              <option :value="null">一级分类</option>
              <option
                v-for="p in parentOptions.filter((x) => x.id !== root.id)"
                :key="p.id"
                :value="p.id"
              >
                {{ p.label }}
              </option>
            </select>
            <button type="button" class="c-btn c-btn--primary" @click="saveEdit">保存</button>
            <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="resetEdit">取消</button>
          </div>
          <div class="form-icon">
            <span class="form-icon__label">图标</span>
            <CategoryIconPicker v-model="form.icon" />
          </div>
        </div>

        <!-- Children -->
        <div v-if="childCount(root) > 0 && isOpen(root.id)" class="cat-children">
          <div class="cat-children__spine" aria-hidden="true" />
          <div
            v-for="(ch, ci) in root.children"
            :key="ch.id"
            class="cat-child-wrap"
          >
            <div
              class="cat-node cat-node--child"
              :class="{
                'cat-node--drag-over': dragOverId === ch.id,
                'cat-node--dragging': dragFrom?.id === ch.id,
                'cat-node--editing': editingId === ch.id,
                'cat-node--last': ci === (root.children?.length || 0) - 1,
              }"
              @dragover="onDragOver(ch, $event)"
              @dragleave="onDragLeave(ch)"
              @drop="onDrop(ch, $event)"
            >
              <span class="cat-child-branch" aria-hidden="true" />
              <div
                class="cat-node__main"
                :draggable="!sorting"
                @dragstart="onDragStart(ch, $event)"
                @dragend="onDragEnd"
              >
                <span class="cat-twist cat-twist--leaf" aria-hidden="true" />
                <span class="cat-node__grip" title="拖拽排序" aria-hidden="true">
                  <AppIcon name="grip-vertical" :size="14" />
                </span>
                <span class="cat-node__icon cat-node__icon--sm">
                  <AppIcon :name="iconForCategory(ch.icon, ch.id)" :size="16" />
                </span>
                <div class="cat-node__body">
                  <div class="cat-node__title">{{ ch.name }}</div>
                  <div class="cat-node__chips">
                    <span class="chip">{{ linkCount(ch) }} 链接</span>
                    <span class="chip chip--mute">展示 {{ ch.display_limit }}</span>
                  </div>
                </div>
              </div>
              <div class="cat-node__sort" @mousedown="armNoDrag" @click.stop>
                <button
                  type="button"
                  class="sort-mini"
                  title="置顶"
                  :disabled="sorting || sibIndex(ch) <= 0"
                  @click="movePin(ch, 'top')"
                >
                  ⤒
                </button>
                <button
                  type="button"
                  class="sort-mini"
                  title="上移"
                  :disabled="sorting || sibIndex(ch) <= 0"
                  @click="moveSibling(ch, -1)"
                >
                  ↑
                </button>
                <button
                  type="button"
                  class="sort-mini"
                  title="下移"
                  :disabled="sorting || sibIndex(ch) >= sibCount(ch) - 1"
                  @click="moveSibling(ch, 1)"
                >
                  ↓
                </button>
                <button
                  type="button"
                  class="sort-mini"
                  title="置底"
                  :disabled="sorting || sibIndex(ch) >= sibCount(ch) - 1"
                  @click="movePin(ch, 'bottom')"
                >
                  ⤓
                </button>
              </div>
              <div class="cat-node__actions" @mousedown="armNoDrag" @click.stop>
                <button
                  type="button"
                  class="c-btn c-btn--ghost c-btn--sm"
                  :class="{ 'c-btn--active-edit': editingId === ch.id }"
                  @click="startEdit(ch)"
                >
                  {{ editingId === ch.id ? '收起' : '编辑' }}
                </button>
                <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="remove(ch.id)">删除</button>
              </div>
            </div>

            <div
              v-if="editingId === ch.id"
              class="cat-inline-edit cat-inline-edit--child"
              @mousedown="armNoDrag"
              @click.stop
            >
              <div class="cat-inline-edit__head">
                <span class="cat-inline-edit__badge">编辑</span>
                <span class="cat-inline-edit__name">{{ ch.name }}</span>
              </div>
              <div class="form-row">
                <input v-model="form.name" class="c-input" placeholder="名称" style="max-width: 200px" />
                <input
                  v-model.number="form.display_limit"
                  class="c-input"
                  type="number"
                  min="1"
                  style="max-width: 100px"
                  title="首页展示条数"
                />
                <select v-model="form.parent_id" class="c-input" style="max-width: 200px">
                  <option :value="null">一级分类</option>
                  <option
                    v-for="p in parentOptions.filter((x) => x.id !== ch.id)"
                    :key="p.id"
                    :value="p.id"
                  >
                    {{ p.label }}
                  </option>
                </select>
                <button type="button" class="c-btn c-btn--primary" @click="saveEdit">保存</button>
                <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="resetEdit">取消</button>
              </div>
              <div class="form-icon">
                <span class="form-icon__label">图标</span>
                <CategoryIconPicker v-model="form.icon" />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.form-card {
  margin-bottom: 4px;
}
.form-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  margin-bottom: 12px;
}
.form-icon {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.form-icon__label {
  font-size: 12px;
  color: var(--console-text-3);
}

/* Inline edit panel under a category node */
.cat-inline-edit {
  margin: 0 10px 12px;
  padding: 12px 14px 14px;
  border-radius: 10px;
  border: 1px solid color-mix(in srgb, var(--console-primary) 35%, var(--console-border));
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--console-primary) 8%, transparent), transparent 40%),
    color-mix(in srgb, var(--console-bg) 50%, var(--console-surface));
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--console-primary) 6%, transparent);
}
.cat-inline-edit--child {
  margin: 0 0 8px 8px;
}
.cat-inline-edit__head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}
.cat-inline-edit__badge {
  font-size: 10px;
  font-weight: 650;
  letter-spacing: 0.08em;
  padding: 2px 7px;
  border-radius: 4px;
  color: var(--console-primary);
  border: 1px solid color-mix(in srgb, var(--console-primary) 40%, transparent);
  background: color-mix(in srgb, var(--console-primary) 12%, transparent);
}
.cat-inline-edit__name {
  font-size: 13px;
  font-weight: 600;
  color: var(--console-text-2);
}
.cat-inline-edit .form-row {
  margin-bottom: 10px;
}
.cat-child-wrap {
  display: flex;
  flex-direction: column;
}
.c-btn--active-edit {
  color: var(--console-primary) !important;
  border-color: color-mix(in srgb, var(--console-primary) 45%, transparent) !important;
  background: color-mix(in srgb, var(--console-primary) 12%, transparent) !important;
}

.tree-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin: 4px 0 8px;
}
.tree-search {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 200px;
  max-width: 360px;
  color: var(--console-text-3);
}
.tree-search .c-input {
  flex: 1;
}
.tree-meta {
  font-size: 12px;
  color: var(--console-text-3);
  font-family: var(--console-mono, ui-monospace, monospace);
}

/* —— Tree shell —— */
.cat-tree {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.cat-tree--busy {
  opacity: 0.72;
  pointer-events: none;
}

.cat-branch {
  border-radius: 10px;
  border: 1px solid var(--console-border);
  background: var(--console-surface);
  overflow: hidden;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.cat-branch--open {
  border-color: color-mix(in srgb, var(--console-primary) 28%, var(--console-border));
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--console-primary) 8%, transparent);
}

/* Node row */
.cat-node {
  position: relative;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 10px;
  padding: 10px 12px;
  min-height: 52px;
  background: transparent;
  transition: background 0.12s;
}
.cat-node--root {
  gap: 8px;
  background: linear-gradient(90deg, color-mix(in srgb, var(--console-primary) 6%, transparent), transparent 48%);
}
.cat-node--root:hover {
  background: linear-gradient(90deg, color-mix(in srgb, var(--console-primary) 11%, transparent), transparent 55%);
}
.cat-node__main {
  display: flex;
  align-items: center;
  gap: 8px 10px;
  flex: 1;
  min-width: 0;
  user-select: none;
}
.cat-node--expandable .cat-node__main {
  cursor: pointer;
}
.cat-node__main[draggable='true'] {
  cursor: grab;
}
.cat-node__main[draggable='true']:active {
  cursor: grabbing;
}
.cat-node--child {
  padding-left: 8px;
  background: color-mix(in srgb, var(--console-bg) 55%, var(--console-surface));
}
.cat-node--child:hover {
  background: color-mix(in srgb, var(--console-primary) 6%, var(--console-surface));
}
.cat-node--editing {
  outline: 1px solid color-mix(in srgb, var(--console-primary) 45%, transparent);
  outline-offset: -1px;
}
.cat-node--dragging {
  opacity: 0.4;
}
.cat-node--drag-over {
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--console-primary) 55%, transparent);
}

/* Left energy rail on roots */
.cat-node__rail {
  position: absolute;
  left: 0;
  top: 10px;
  bottom: 10px;
  width: 3px;
  border-radius: 0 2px 2px 0;
  background: color-mix(in srgb, var(--console-primary) 55%, transparent);
  opacity: 0.55;
}
.cat-branch--open > .cat-node--root .cat-node__rail {
  opacity: 1;
  box-shadow: 0 0 10px color-mix(in srgb, var(--console-primary) 35%, transparent);
}

/* Custom expand control — not browser default triangle */
.cat-twist {
  width: 28px;
  height: 28px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  border: 1px solid color-mix(in srgb, var(--console-primary) 35%, var(--console-border));
  border-radius: 6px;
  background: color-mix(in srgb, var(--console-primary) 8%, var(--console-surface-2, #1c2330));
  color: var(--console-primary);
  padding: 0;
  transition:
    background 0.15s,
    border-color 0.15s,
    transform 0.15s,
    box-shadow 0.15s;
}
.cat-twist:hover {
  background: color-mix(in srgb, var(--console-primary) 16%, transparent);
  border-color: color-mix(in srgb, var(--console-primary) 55%, transparent);
  box-shadow: 0 0 12px color-mix(in srgb, var(--console-primary) 18%, transparent);
}
.cat-twist__glyph {
  display: block;
  width: 0;
  height: 0;
  border-style: solid;
  border-width: 5px 0 5px 7px;
  border-color: transparent transparent transparent currentColor;
  margin-left: 2px;
  transition: transform 0.18s ease;
  transform-origin: 40% 50%;
}
.cat-twist--open .cat-twist__glyph {
  transform: rotate(90deg);
}
.cat-twist--leaf {
  border-color: transparent;
  background: transparent;
  pointer-events: none;
  box-shadow: none;
}
.cat-twist--leaf::after {
  content: '';
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: color-mix(in srgb, var(--console-text-3) 50%, transparent);
}

.cat-node__grip {
  display: grid;
  place-items: center;
  width: 16px;
  color: var(--console-text-3);
  opacity: 0.45;
  flex-shrink: 0;
}
.cat-node:hover .cat-node__grip {
  opacity: 0.9;
  color: var(--console-primary);
}

.cat-node__icon {
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  border-radius: 9px;
  color: var(--console-primary);
  background: color-mix(in srgb, var(--console-primary) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--console-primary) 22%, transparent);
}
.cat-node__icon--sm {
  width: 30px;
  height: 30px;
  border-radius: 8px;
}

.cat-node__body {
  flex: 1;
  min-width: 120px;
}
.cat-node__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--console-text);
  letter-spacing: 0.01em;
}
.cat-node__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 4px;
}
.chip {
  display: inline-flex;
  align-items: center;
  height: 20px;
  padding: 0 7px;
  border-radius: 4px;
  font-size: 11px;
  font-family: var(--console-mono, ui-monospace, monospace);
  color: var(--console-text-2);
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--console-border);
}
.chip--sub {
  color: color-mix(in srgb, var(--console-primary) 90%, #fff);
  border-color: color-mix(in srgb, var(--console-primary) 30%, transparent);
  background: color-mix(in srgb, var(--console-primary) 10%, transparent);
}
.chip--mute {
  color: var(--console-text-3);
}

.cat-node__sort {
  display: inline-flex;
  gap: 2px;
  flex-shrink: 0;
}
.sort-mini {
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  border: 1px solid var(--console-border);
  border-radius: 5px;
  background: transparent;
  color: var(--console-text-2);
  font-size: 12px;
  line-height: 1;
  padding: 0;
  cursor: pointer;
}
.sort-mini:hover:not(:disabled) {
  color: var(--console-primary);
  border-color: color-mix(in srgb, var(--console-primary) 40%, transparent);
}
.sort-mini:disabled {
  opacity: 0.28;
  cursor: not-allowed;
}

.cat-node__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  flex-shrink: 0;
}

/* Children panel */
.cat-children {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 0;
  padding: 0 10px 10px 22px;
  border-top: 1px solid color-mix(in srgb, var(--console-border) 80%, transparent);
  background: color-mix(in srgb, var(--console-bg) 40%, transparent);
}
.cat-children__spine {
  position: absolute;
  left: 34px;
  top: 0;
  bottom: 18px;
  width: 2px;
  border-radius: 1px;
  background: linear-gradient(
    180deg,
    color-mix(in srgb, var(--console-primary) 35%, transparent),
    color-mix(in srgb, var(--console-primary) 8%, transparent)
  );
  opacity: 0.7;
}
.cat-node--child {
  border-radius: 8px;
  margin-top: 6px;
  border: 1px solid color-mix(in srgb, var(--console-border) 70%, transparent);
}
.cat-child-branch {
  position: absolute;
  left: -14px;
  top: 50%;
  width: 12px;
  height: 2px;
  background: color-mix(in srgb, var(--console-primary) 28%, transparent);
  border-radius: 1px;
}

@media (max-width: 900px) {
  .cat-node {
    gap: 8px;
  }
  .cat-node__sort {
    order: 5;
    width: 100%;
    justify-content: flex-start;
    padding-left: 36px;
  }
  .cat-node__actions {
    margin-left: auto;
  }
}
</style>
