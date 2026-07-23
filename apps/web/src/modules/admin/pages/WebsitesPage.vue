<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { apiGet, apiPost, apiDelete, apiPatch } from '@/shared/api/client'
import { useToast } from '@/shared/composables/useToast'
import type { Category, Website } from '@/shared/types/models'
import AdminTable from '../components/AdminTable.vue'

const toast = useToast()
const items = ref<Website[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const q = ref('')
const categoryId = ref<number | null>(null)
const categories = ref<Category[]>([])
const loading = ref(false)
const selected = ref<number[]>([])

const form = ref({
  title: '',
  url: '',
  description: '',
  category_id: null as number | null,
  is_private: false,
  is_featured: false,
})
const editingId = ref<number | null>(null)
const showForm = ref(false)

function flatCats(list: Category[], depth = 0): { id: number; label: string }[] {
  const out: { id: number; label: string }[] = []
  for (const c of list) {
    out.push({ id: c.id, label: `${'— '.repeat(depth)}${c.name}` })
    if (c.children?.length) out.push(...flatCats(c.children, depth + 1))
  }
  return out
}

async function load() {
  loading.value = true
  try {
    let url = `/api/v1/admin/websites?page=${page.value}&page_size=${pageSize}`
    if (q.value.trim()) url += `&q=${encodeURIComponent(q.value.trim())}`
    if (categoryId.value) url += `&category_id=${categoryId.value}`
    const data = await apiGet<{ items: Website[]; total: number }>(url)
    items.value = data.items || []
    total.value = data.total
    selected.value = []
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

async function loadCats() {
  categories.value = await apiGet('/api/v1/admin/categories')
}

function letter(t: string) {
  return (t || '?').trim().slice(0, 1).toUpperCase()
}

function openCreate() {
  editingId.value = null
  form.value = {
    title: '',
    url: '',
    description: '',
    category_id: categoryId.value,
    is_private: false,
    is_featured: false,
  }
  showForm.value = true
}

function openEdit(w: Website) {
  editingId.value = w.id
  form.value = {
    title: w.title,
    url: w.url,
    description: w.description || '',
    category_id: w.category_id ?? null,
    is_private: w.is_private,
    is_featured: w.is_featured,
  }
  showForm.value = true
}

async function save() {
  try {
    const payload = { ...form.value, force: true }
    if (editingId.value) {
      await apiPatch(`/api/v1/admin/websites/${editingId.value}`, payload)
      toast.success('已保存')
    } else {
      await apiPost('/api/v1/admin/websites', payload)
      toast.success('已添加')
    }
    showForm.value = false
    await load()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '保存失败')
  }
}

async function removeOne(id: number) {
  if (!confirm('确认删除该链接？')) return
  await apiDelete(`/api/v1/admin/websites/${id}`)
  toast.success('已删除')
  await load()
}

async function batchDelete() {
  if (!selected.value.length) return
  if (!confirm(`批量删除 ${selected.value.length} 条？`)) return
  await apiPost('/api/v1/admin/websites/batch-delete', { ids: selected.value })
  toast.success('批量删除完成')
  await load()
}

function toggleAll(e: Event) {
  const on = (e.target as HTMLInputElement).checked
  selected.value = on ? items.value.map((i) => i.id) : []
}

function toggleOne(id: number, e: Event) {
  const on = (e.target as HTMLInputElement).checked
  if (on) selected.value = [...selected.value, id]
  else selected.value = selected.value.filter((x) => x !== id)
}

const totalPages = () => Math.max(1, Math.ceil(total.value / pageSize))

watch([page, categoryId], () => load())

onMounted(async () => {
  await loadCats()
  await load()
})
</script>

<template>
  <div>
    <header class="page-header">
      <div>
        <h1>链接管理</h1>
        <p>管理全站导航链接。长文本在表格内单行省略，悬停可看完整内容。共 {{ total }} 条。</p>
      </div>
      <div class="page-header__actions">
        <button type="button" class="c-btn c-btn--primary" @click="openCreate">添加链接</button>
      </div>
    </header>

    <div class="c-toolbar">
      <input
        v-model="q"
        class="c-input"
        placeholder="搜索标题 / URL / 描述"
        @keyup.enter="page = 1; load()"
      />
      <select v-model="categoryId" class="c-input" @change="page = 1">
        <option :value="null">全部分类</option>
        <option v-for="c in flatCats(categories)" :key="c.id" :value="c.id">{{ c.label }}</option>
      </select>
      <button type="button" class="c-btn" @click="page = 1; load()">搜索</button>
      <button v-if="selected.length" type="button" class="c-btn c-btn--danger" @click="batchDelete">
        删除所选 ({{ selected.length }})
      </button>
    </div>

    <div v-if="showForm" class="c-card c-card__body" style="margin-bottom: 14px">
      <h3 class="c-card__title">{{ editingId ? '编辑链接' : '添加链接' }}</h3>
      <div class="c-form-2">
        <label>标题 <input v-model="form.title" class="c-input" placeholder="网站标题" /></label>
        <label>URL <input v-model="form.url" class="c-input" placeholder="https://" /></label>
        <label class="span-2">
          描述
          <input v-model="form.description" class="c-input" placeholder="可选" />
        </label>
        <label>
          分类
          <select v-model="form.category_id" class="c-input">
            <option :value="null">未分类</option>
            <option v-for="c in flatCats(categories)" :key="c.id" :value="c.id">{{ c.label }}</option>
          </select>
        </label>
        <div class="c-check-row">
          <label><input v-model="form.is_private" type="checkbox" /> 私有</label>
          <label><input v-model="form.is_featured" type="checkbox" /> 精选</label>
        </div>
      </div>
      <div style="display: flex; gap: 8px; margin-top: 14px">
        <button type="button" class="c-btn c-btn--primary" @click="save">保存</button>
        <button type="button" class="c-btn c-btn--ghost" @click="showForm = false">取消</button>
      </div>
    </div>

    <AdminTable :loading="loading" :is-empty="!items.length" empty="暂无链接">
      <template #head>
        <tr>
          <th class="c-col-check"><input type="checkbox" @change="toggleAll" /></th>
          <th class="c-col-icon" />
          <th>标题 / 地址</th>
          <th class="c-col-cat">分类</th>
          <th class="c-col-status">状态</th>
          <th class="c-col-actions">操作</th>
        </tr>
      </template>
      <tr v-for="w in items" :key="w.id">
        <td class="c-col-check">
          <input
            type="checkbox"
            :checked="selected.includes(w.id)"
            @change="toggleOne(w.id, $event)"
          />
        </td>
        <td class="c-col-icon">
          <img
            v-if="w.icon"
            class="c-avatar-sm"
            :src="w.icon"
            alt=""
            loading="lazy"
            referrerpolicy="no-referrer"
          />
          <span v-else class="c-avatar-fallback">{{ letter(w.title) }}</span>
        </td>
        <td>
          <div class="c-cell-title" :title="w.title">{{ w.title }}</div>
          <span class="c-cell-sub" :title="w.url">{{ w.url }}</span>
        </td>
        <td class="c-col-cat">
          <div class="c-cell-ellipsis" :title="w.category_name || ''">{{ w.category_name || '—' }}</div>
        </td>
        <td class="c-col-status">
          <span v-if="w.is_private" class="c-tag c-tag--warn">私有</span>
          <span v-else class="c-tag c-tag--ok">公开</span>
          <span v-if="!w.is_valid" class="c-tag c-tag--danger">失效</span>
        </td>
        <td class="c-col-actions">
          <div class="c-cell-actions">
            <button type="button" class="c-btn c-btn--sm c-btn--ghost" @click="openEdit(w)">编辑</button>
            <button type="button" class="c-btn c-btn--sm c-btn--ghost" @click="removeOne(w.id)">删除</button>
          </div>
        </td>
      </tr>
      <template #footer>
        <span>第 {{ page }} / {{ totalPages() }} 页 · {{ total }} 条</span>
        <div class="c-pagination__btns">
          <button type="button" class="c-btn c-btn--sm" :disabled="page <= 1" @click="page--">上一页</button>
          <button type="button" class="c-btn c-btn--sm" :disabled="page >= totalPages()" @click="page++">
            下一页
          </button>
        </div>
      </template>
    </AdminTable>
  </div>
</template>
