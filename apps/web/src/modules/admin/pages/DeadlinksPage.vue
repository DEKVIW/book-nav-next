<script setup lang="ts">
/**
 * 死链检测：启动检测 + 查看失效结果
 * 任务进度也可在任务中心统一查看
 */
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { apiGet, apiPost } from '@/shared/api/client'
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
  result_json?: string
}

interface DeadItem {
  id: number
  website_id: number
  url: string
  is_valid: boolean
  status_code?: number
  error_type?: string
  website_title?: string
  batch_id?: string
}

const toast = useToast()
const jobs = ref<Job[]>([])
const results = ref<DeadItem[]>([])
const batchId = ref('')
const loading = ref(false)
const polling = ref(false)

async function loadJobs() {
  jobs.value = ((await apiGet('/api/v1/admin/jobs')) || []).filter(
    (j: Job) => j.type === 'deadlink_check',
  )
  const last = jobs.value[0]
  if (last?.result_json) {
    try {
      const r = JSON.parse(last.result_json)
      if (r.batch_id) batchId.value = r.batch_id
    } catch {
      /* ignore */
    }
  }
}

async function start() {
  try {
    const j = await apiPost<Job>('/api/v1/admin/jobs/deadlink')
    toast.success(`任务 #${j.id} 已启动`)
    polling.value = true
    poll(j.id)
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '启动失败')
  }
}

async function poll(id: number) {
  const tick = async () => {
    try {
      const j = await apiGet<Job>(`/api/v1/admin/jobs/${id}`)
      await loadJobs()
      if (j.status === 'completed' || j.status === 'failed') {
        polling.value = false
        if (j.result_json) {
          try {
            const r = JSON.parse(j.result_json)
            if (r.batch_id) {
              batchId.value = r.batch_id
              await loadResults()
            }
          } catch {
            /* ignore */
          }
        }
        toast.success(j.status === 'completed' ? '检测完成' : '检测失败')
        return
      }
      setTimeout(tick, 1500)
    } catch {
      polling.value = false
    }
  }
  tick()
}

async function loadResults() {
  if (!batchId.value) return
  loading.value = true
  try {
    results.value = await apiGet(
      `/api/v1/admin/deadlinks?batch_id=${encodeURIComponent(batchId.value)}&invalid_only=1`,
    )
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '加载结果失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await loadJobs()
  if (batchId.value) await loadResults()
})
</script>

<template>
  <div class="admin-page">
    <header class="page-header">
      <h1>死链检测</h1>
      <div class="page-header__actions">
        <RouterLink class="c-btn c-btn--ghost c-btn--sm" to="/admin/jobs">任务中心</RouterLink>
        <button
          type="button"
          class="c-btn c-btn--ghost c-btn--sm"
          :disabled="!batchId"
          @click="loadResults"
        >
          刷新失效列表
        </button>
        <button type="button" class="c-btn c-btn--primary" :disabled="polling" @click="start">
          {{ polling ? '检测中…' : '开始检测' }}
        </button>
      </div>
    </header>

    <section v-if="jobs.length" class="c-card c-card__body">
      <h3 class="c-card__title">最近检测</h3>
      <div class="job-list">
        <div v-for="j in jobs.slice(0, 5)" :key="j.id" class="job-line">
          <span class="mono">#{{ j.id }}</span>
          <span class="c-tag">{{ j.status }}</span>
          <span class="mono">{{ j.progress }}/{{ j.total }}</span>
          <span class="mono">ok {{ j.success }} / fail {{ j.failed }}</span>
        </div>
      </div>
    </section>

    <AdminTable :loading="loading" :is-empty="!results.length" empty="暂无失效链接（先运行检测）">
      <template #head>
        <tr>
          <th>标题</th>
          <th>URL</th>
          <th style="width: 88px">状态码</th>
          <th style="width: 120px">错误</th>
        </tr>
      </template>
      <tr v-for="r in results" :key="r.id">
        <td>
          <div class="c-cell-title" :title="r.website_title">{{ r.website_title || '—' }}</div>
        </td>
        <td>
          <div class="c-cell-ellipsis" :title="r.url">{{ r.url }}</div>
        </td>
        <td class="mono">{{ r.status_code ?? '—' }}</td>
        <td>
          <div class="c-cell-ellipsis">{{ r.error_type || '—' }}</div>
        </td>
      </tr>
    </AdminTable>
  </div>
</template>

<style scoped>
.job-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.job-line {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  font-size: 12px;
  color: var(--console-text-2);
  padding: 8px 0;
  border-bottom: 1px solid var(--console-border);
}
.job-line:last-child {
  border-bottom: 0;
}
.mono {
  font-family: var(--font-mono, ui-monospace, monospace);
}
</style>
