<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { apiGet, apiPost, apiDelete, apiPatch } from '@/shared/api/client'
import { useToast } from '@/shared/composables/useToast'
import type { Category } from '@/shared/types/models'
import AdminTable from '../components/AdminTable.vue'

const toast = useToast()
const categories = ref<Category[]>([])
const loading = ref(false)
const form = ref({
  name: '',
  color: '#3DE7FF',
  description: '',
  display_limit: 10,
  parent_id: null as number | null,
})
const editingId = ref<number | null>(null)

function flat(list: Category[], depth = 0): (Category & { _label: string; _depth: number })[] {
  const out: (Category & { _label: string; _depth: number })[] = []
  for (const c of list) {
    out.push({ ...c, _label: `${'— '.repeat(depth)}${c.name}`, _depth: depth })
    if (c.children?.length) out.push(...flat(c.children, depth + 1))
  }
  return out
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

function startEdit(c: Category) {
  editingId.value = c.id
  form.value = {
    name: c.name,
    color: c.color || '#3DE7FF',
    description: c.description || '',
    display_limit: c.display_limit || 10,
    parent_id: c.parent_id ?? null,
  }
}

function resetForm() {
  editingId.value = null
  form.value = {
    name: '',
    color: '#3DE7FF',
    description: '',
    display_limit: 10,
    parent_id: null,
  }
}

async function save() {
  if (!form.value.name.trim()) {
    toast.error('请填写名称')
    return
  }
  try {
    if (editingId.value) {
      await apiPatch(`/api/v1/admin/categories/${editingId.value}`, form.value)
      toast.success('已更新')
    } else {
      await apiPost('/api/v1/admin/categories', form.value)
      toast.success('已创建')
    }
    resetForm()
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
    await load()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '删除失败')
  }
}

onMounted(load)
</script>

<template>
  <div>
    <header class="page-header">
      <div>
        <h1>分类管理</h1>
        <p>树形分类 · 显示数量与首页 display_limit</p>
      </div>
    </header>

    <div class="c-card c-card__body" style="margin-bottom: 14px">
      <h3 class="c-card__title">{{ editingId ? '编辑分类' : '新建分类' }}</h3>
      <div style="display: flex; flex-wrap: wrap; gap: 8px; align-items: center">
        <input v-model="form.name" class="c-input" placeholder="名称" style="max-width: 200px" />
        <input v-model="form.color" class="c-input" type="color" style="max-width: 56px; padding: 2px" />
        <input
          v-model.number="form.display_limit"
          class="c-input"
          type="number"
          min="1"
          style="max-width: 100px"
          title="首页展示条数"
        />
        <select v-model="form.parent_id" class="c-input" style="max-width: 180px">
          <option :value="null">一级分类</option>
          <option v-for="c in flat(categories)" :key="c.id" :value="c.id">{{ c._label }}</option>
        </select>
        <button type="button" class="c-btn c-btn--primary" @click="save">
          {{ editingId ? '保存' : '创建' }}
        </button>
        <button v-if="editingId" type="button" class="c-btn c-btn--ghost c-btn--sm" @click="resetForm">取消</button>
      </div>
    </div>

    <AdminTable :loading="loading" :is-empty="!flat(categories).length" empty="暂无分类">
      <template #head>
        <tr>
          <th style="width: 48px">色</th>
          <th>名称</th>
          <th style="width: 100px">链接数</th>
          <th style="width: 100px">展示上限</th>
          <th class="c-col-actions">操作</th>
        </tr>
      </template>
      <tr v-for="c in flat(categories)" :key="c.id">
        <td>
          <span class="swatch" :style="{ background: c.color || 'var(--energy)' }" />
        </td>
        <td>
          <div class="c-cell-title" :style="{ paddingLeft: `${c._depth * 14}px` }">{{ c.name }}</div>
        </td>
        <td>{{ c.website_count ?? c.direct_count ?? 0 }}</td>
        <td>{{ c.display_limit }}</td>
        <td class="c-col-actions">
          <div class="c-cell-actions">
            <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="startEdit(c)">编辑</button>
            <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="remove(c.id)">删除</button>
          </div>
        </td>
      </tr>
    </AdminTable>
  </div>
</template>

<style scoped>
.swatch {
  display: inline-block;
  width: 16px;
  height: 16px;
  border-radius: 2px;
  border: 1px solid var(--stroke-dim);
}
</style>
