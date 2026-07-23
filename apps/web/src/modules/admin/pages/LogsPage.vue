<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { apiDelete, apiGet, apiPost } from '@/shared/api/client'
import { useToast } from '@/shared/composables/useToast'
import AdminTable from '../components/AdminTable.vue'

interface OpLog {
  id: number
  action: string
  website_title?: string
  website_url?: string
  category_name?: string
  user_id?: number
  created_at?: string
}

const toast = useToast()
const items = ref<OpLog[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const data = await apiGet<{ items: OpLog[]; total: number }>(
      `/api/v1/admin/operation-logs?page=${page.value}&page_size=30`,
    )
    items.value = data.items || []
    total.value = data.total
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

async function remove(id: number) {
  if (!confirm(`删除日志 #${id}？`)) return
  try {
    await apiDelete(`/api/v1/admin/operation-logs/${id}`)
    toast.success('已删除')
    await load()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '删除失败')
  }
}

async function clearAll() {
  if (!confirm('清空全部操作日志？')) return
  try {
    await apiPost('/api/v1/admin/operation-logs/clear')
    toast.success('已清空')
    page.value = 1
    await load()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '清空失败')
  }
}

function pages() {
  return Math.max(1, Math.ceil(total.value / 30))
}

onMounted(load)
</script>

<template>
  <div class="admin-page">
    <header class="page-header">
      <div>
        <h1>操作日志</h1>
        <p>共 {{ total }} 条</p>
      </div>
      <div class="page-header__actions">
        <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="clearAll">清空</button>
        <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="load">刷新</button>
      </div>
    </header>

    <AdminTable :loading="loading" :is-empty="!items.length" empty="暂无日志">
      <template #head>
        <tr>
          <th class="c-col-id">ID</th>
          <th style="width: 80px">动作</th>
          <th>链接</th>
          <th class="c-col-cat">分类</th>
          <th class="c-col-date">时间</th>
          <th class="c-col-actions">操作</th>
        </tr>
      </template>
      <tr v-for="l in items" :key="l.id">
        <td>{{ l.id }}</td>
        <td>
          <span class="c-tag" :class="{ 'tag--danger': l.action === 'DELETE', 'tag--ok': l.action === 'ADD' }">
            {{ l.action }}
          </span>
        </td>
        <td>
          <div class="c-cell-title" :title="l.website_title">{{ l.website_title || '—' }}</div>
          <span class="c-cell-sub" :title="l.website_url">{{ l.website_url }}</span>
        </td>
        <td class="c-col-cat">
          <div class="c-cell-ellipsis">{{ l.category_name || '—' }}</div>
        </td>
        <td class="c-col-date">
          <div class="c-cell-ellipsis" :title="l.created_at">{{ l.created_at?.slice(0, 19) }}</div>
        </td>
        <td class="c-col-actions">
          <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="remove(l.id)">删除</button>
        </td>
      </tr>
      <template #footer>
        <span>第 {{ page }} / {{ pages() }} 页</span>
        <div class="c-pagination__btns">
          <button type="button" class="c-btn c-btn--ghost c-btn--sm" :disabled="page <= 1" @click="page--; load()">
            上一页
          </button>
          <button
            type="button"
            class="c-btn c-btn--ghost c-btn--sm"
            :disabled="page >= pages()"
            @click="page++; load()"
          >
            下一页
          </button>
        </div>
      </template>
    </AdminTable>
  </div>
</template>
