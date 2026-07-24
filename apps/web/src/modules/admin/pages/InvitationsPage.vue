<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { apiGet, apiPost, apiDelete } from '@/shared/api/client'
import { useToast } from '@/shared/composables/useToast'
import AdminTable from '../components/AdminTable.vue'

interface Invite {
  id: number
  code: string
  is_active: boolean
  created_at?: string
  used_by?: number | null
}

const toast = useToast()
const items = ref<Invite[]>([])
const loading = ref(false)
const count = ref(1)

async function load() {
  loading.value = true
  try {
    items.value = await apiGet('/api/v1/admin/invitations')
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

async function generate() {
  try {
    const list = await apiPost<Invite[]>('/api/v1/admin/invitations', { count: count.value })
    toast.success(`已生成 ${list.length} 个邀请码`)
    await load()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '失败')
  }
}

async function remove(id: number) {
  if (!confirm('删除该邀请码？')) return
  await apiDelete(`/api/v1/admin/invitations/${id}`)
  toast.success('已删除')
  await load()
}

async function copy(code: string) {
  try {
    await navigator.clipboard.writeText(code)
    toast.success('已复制')
  } catch {
    toast.error('复制失败')
  }
}

onMounted(load)
</script>

<template>
  <div>
    <header class="page-header">
      <h1>邀请码</h1>
      <div class="page-header__actions">
        <input v-model.number="count" class="c-input" type="number" min="1" max="50" style="width: 80px" />
        <button type="button" class="c-btn c-btn--primary" @click="generate">生成</button>
      </div>
    </header>

    <AdminTable :loading="loading" :is-empty="!items.length" empty="暂无邀请码">
      <template #head>
        <tr>
          <th class="c-col-id">ID</th>
          <th>邀请码</th>
          <th style="width: 100px">状态</th>
          <th class="c-col-actions">操作</th>
        </tr>
      </template>
      <tr v-for="i in items" :key="i.id">
        <td>{{ i.id }}</td>
        <td>
          <code class="code">{{ i.code }}</code>
        </td>
        <td>
          <span v-if="i.is_active" class="c-tag c-tag--ok">可用</span>
          <span v-else class="c-tag">已用</span>
        </td>
        <td class="c-col-actions">
          <div class="c-cell-actions">
            <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="copy(i.code)">复制</button>
            <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="remove(i.id)">删除</button>
          </div>
        </td>
      </tr>
    </AdminTable>
  </div>
</template>

<style scoped>
.code {
  font-family: var(--font-mono);
  color: var(--energy);
  font-size: 13px;
}
</style>
