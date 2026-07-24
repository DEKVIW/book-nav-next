<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { apiGet, apiPost, apiDelete, apiPatch } from '@/shared/api/client'
import { useToast } from '@/shared/composables/useToast'
import type { Category } from '@/shared/types/models'
import AdminTable from '../components/AdminTable.vue'
import CategoryIconPicker from '../components/CategoryIconPicker.vue'
import AppIcon from '@/shared/ui/AppIcon.vue'
import { iconForCategory } from '@/shared/icons/registry'

const toast = useToast()
const categories = ref<Category[]>([])
const loading = ref(false)
const form = ref({
  name: '',
  icon: 'folder',
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
    icon: iconForCategory(c.icon, c.id),
    description: c.description || '',
    display_limit: c.display_limit || 10,
    parent_id: c.parent_id ?? null,
  }
}

function resetForm() {
  editingId.value = null
  form.value = {
    name: '',
    icon: 'folder',
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
  const payload = {
    name: form.value.name.trim(),
    icon: form.value.icon || 'folder',
    description: form.value.description,
    display_limit: form.value.display_limit,
    parent_id: form.value.parent_id,
  }
  try {
    if (editingId.value) {
      await apiPatch(`/api/v1/admin/categories/${editingId.value}`, payload)
      toast.success('已更新')
    } else {
      await apiPost('/api/v1/admin/categories', payload)
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
      <h1>分类管理</h1>
    </header>

    <div class="c-card c-card__body form-card">
      <h3 class="c-card__title">{{ editingId ? '编辑分类' : '新建分类' }}</h3>
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
        <select v-model="form.parent_id" class="c-input" style="max-width: 180px">
          <option :value="null">一级分类</option>
          <option v-for="c in flat(categories)" :key="c.id" :value="c.id">{{ c._label }}</option>
        </select>
        <button type="button" class="c-btn c-btn--primary" @click="save">
          {{ editingId ? '保存' : '创建' }}
        </button>
        <button v-if="editingId" type="button" class="c-btn c-btn--ghost c-btn--sm" @click="resetForm">
          取消
        </button>
      </div>
      <div class="form-icon">
        <span class="form-icon__label">图标</span>
        <CategoryIconPicker v-model="form.icon" />
      </div>
    </div>

    <AdminTable :loading="loading" :is-empty="!flat(categories).length" empty="暂无分类">
      <template #head>
        <tr>
          <th style="width: 56px">图标</th>
          <th>名称</th>
          <th style="width: 100px">链接数</th>
          <th style="width: 100px">展示上限</th>
          <th class="c-col-actions">操作</th>
        </tr>
      </template>
      <tr v-for="c in flat(categories)" :key="c.id">
        <td>
          <span class="icon-cell">
            <AppIcon :name="iconForCategory(c.icon, c.id)" :size="18" />
          </span>
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
.form-card {
  margin-bottom: 14px;
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
  color: var(--text-muted);
  letter-spacing: 0.04em;
}
.icon-cell {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  color: var(--energy);
  background: rgba(61, 231, 255, 0.08);
  border: 1px solid rgba(61, 231, 255, 0.2);
  border-radius: 8px;
}
</style>
