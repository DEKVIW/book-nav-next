<script setup lang="ts">
import { onMounted, ref } from 'vue'
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
  jobs.value = (await apiGet('/api/v1/admin/jobs')) || []
  const last = jobs.value.find((j) => j.type === 'deadlink_check')
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

onMounted(loadJobs)
</script>

<template>
  <div>
    <header class="page-header">
      <div>
        <h1>死链检测</h1>
        <p>批量检测链接是否可访问（SSRF 防护已启用）</p>
      </div>
      <div style="display:flex;flex-wrap:wrap;gap:8px;align-items:center">
        <button type="button" class="c-btn c-btn--primary" :disabled="polling" @click="start">
          {{ polling ? '检测中…' : '开始检测' }}
        </button>
        <button type="button" class="c-btn c-btn--ghost c-btn--sm" :disabled="!batchId" @click="loadResults">
          刷新失效列表
        </button>
      </div>
    </header>

    <section class="admin-section admin-card" style="padding: 12px; margin-bottom: 12px">
      <h3 style="margin: 0 0 8px; font-size: 13px">最近任务</h3>
      <div v-if="!jobs.filter((j) => j.type === 'deadlink_check').length" class="c-empty">暂无任务</div>
      <div
        v-for="j in jobs.filter((j) => j.type === 'deadlink_check').slice(0, 5)"
        :key="j.id"
        class="job-line"
      >
        <span>#{{ j.id }}</span>
        <span class="c-tag">{{ j.status }}</span>
        <span>{{ j.progress }}/{{ j.total }}</span>
        <span>ok {{ j.success }} / fail {{ j.failed }}</span>
      </div>
    </section>

    <AdminTable :loading="loading" :is-empty="!results.length" empty="暂无失效链接（先运行检测）">
      <template #head>
        <tr>
          <th>标题</th>
          <th>URL</th>
          <th style="width: 80px">状态码</th>
          <th style="width: 100px">错误</th>
        </tr>
      </template>
      <tr v-for="r in results" :key="r.id">
        <td>
          <div class="c-cell-title" :title="r.website_title">{{ r.website_title || '—' }}</div>
        </td>
        <td>
          <div class="c-cell-ellipsis" :title="r.url">{{ r.url }}</div>
        </td>
        <td>{{ r.status_code ?? '—' }}</td>
        <td>
          <div class="c-cell-ellipsis">{{ r.error_type || '—' }}</div>
        </td>
      </tr>
    </AdminTable>
  </div>
</template>

<style scoped>
.job-line {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  font-size: 12px;
  font-family: var(--font-mono);
  color: var(--text-secondary);
  padding: 6px 0;
  border-bottom: 1px solid var(--stroke-dim);
}
.job-line:last-child {
  border-bottom: 0;
}
</style>
