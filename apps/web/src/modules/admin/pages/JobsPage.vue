<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { apiDelete, apiGet, apiPost } from '@/shared/api/client'
import { useAuthStore } from '@/shared/stores/auth'
import { useToast } from '@/shared/composables/useToast'
import AdminTable from '../components/AdminTable.vue'

interface Job {
  id: number
  type: string
  status: string
  progress: number
  total: number
  success: number
  failed: number
  error?: string
}

const toast = useToast()
const auth = useAuthStore()
const jobs = ref<Job[]>([])
const loading = ref(false)
const starting = ref('')
let timer: ReturnType<typeof setInterval> | null = null

const typeLabel: Record<string, string> = {
  deadlink_check: '死链检测',
  icon_sync: '图标抓取',
  vector_index: '向量索引',
}

async function load() {
  loading.value = true
  try {
    jobs.value = await apiGet('/api/v1/admin/jobs')
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

async function start(kind: 'vector' | 'icons' | 'deadlink') {
  if (!auth.isSuper) {
    toast.error('需要超级管理员')
    return
  }
  starting.value = kind
  try {
    const path =
      kind === 'vector'
        ? '/api/v1/admin/jobs/vector-index'
        : kind === 'icons'
          ? '/api/v1/admin/jobs/icons'
          : '/api/v1/admin/jobs/deadlink'
    const j = await apiPost<Job>(path)
    toast.success(`任务 #${j.id} 已启动`)
    await load()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '启动失败')
  } finally {
    starting.value = ''
  }
}

async function remove(id: number) {
  if (!confirm(`删除任务 #${id}？`)) return
  try {
    await apiDelete(`/api/v1/admin/jobs/${id}`)
    toast.success('已删除')
    await load()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '删除失败')
  }
}

async function clearFinished() {
  if (!confirm('清空所有已完成/失败的任务？')) return
  try {
    const r = await apiPost<{ deleted: number }>('/api/v1/admin/jobs/clear')
    toast.success(`已清理 ${r.deleted ?? 0} 条`)
    await load()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '清理失败')
  }
}

function canDelete(j: Job) {
  return j.status !== 'pending' && j.status !== 'running'
}

onMounted(() => {
  load()
  timer = setInterval(load, 4000)
})
onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <div class="admin-page">
    <header class="page-header">
      <div>
        <h1>任务</h1>
        <p>后台任务进度</p>
      </div>
      <div class="page-header__actions">
        <button
          v-if="auth.isSuper"
          type="button"
          class="c-btn c-btn--primary c-btn--sm"
          :disabled="!!starting"
          @click="start('vector')"
        >
          向量索引
        </button>
        <button
          v-if="auth.isSuper"
          type="button"
          class="c-btn c-btn--ghost c-btn--sm"
          :disabled="!!starting"
          @click="start('icons')"
        >
          图标抓取
        </button>
        <button
          v-if="auth.isSuper"
          type="button"
          class="c-btn c-btn--ghost c-btn--sm"
          :disabled="!!starting"
          @click="start('deadlink')"
        >
          死链检测
        </button>
        <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="clearFinished">清空已完成</button>
        <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="load">刷新</button>
      </div>
    </header>

    <AdminTable :loading="loading" :is-empty="!jobs.length" empty="暂无任务">
      <template #head>
        <tr>
          <th class="c-col-id">ID</th>
          <th>类型</th>
          <th style="width: 100px">状态</th>
          <th style="width: 120px">进度</th>
          <th style="width: 80px">成功</th>
          <th style="width: 80px">失败</th>
          <th>错误</th>
          <th class="c-col-actions">操作</th>
        </tr>
      </template>
      <tr v-for="j in jobs" :key="j.id">
        <td>{{ j.id }}</td>
        <td class="c-cell-ellipsis">{{ typeLabel[j.type] || j.type }}</td>
        <td><span class="c-tag">{{ j.status }}</span></td>
        <td>{{ j.progress }}/{{ j.total }}</td>
        <td>{{ j.success }}</td>
        <td>{{ j.failed }}</td>
        <td>
          <div class="c-cell-ellipsis" :title="j.error">{{ j.error || '—' }}</div>
        </td>
        <td class="c-col-actions">
          <button
            v-if="canDelete(j)"
            type="button"
            class="c-btn c-btn--ghost c-btn--sm"
            @click="remove(j.id)"
          >
            删除
          </button>
          <span v-else class="muted">—</span>
        </td>
      </tr>
    </AdminTable>
  </div>
</template>

<style scoped>
.muted {
  color: var(--console-text-3);
  font-size: 12px;
}
</style>
