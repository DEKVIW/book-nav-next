<script setup lang="ts">
/**
 * 数据管理（Tab）
 * - 导入导出
 * - 清空数据
 * - 死链检测（业务启动 + 结果，任务中心只看进度）
 */
import { onMounted, ref, watch } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { apiDownload, apiGet, apiPost, apiPostForm } from '@/shared/api/client'
import { useToast } from '@/shared/composables/useToast'
import AdminTable from '../components/AdminTable.vue'

type TabKey = 'io' | 'danger' | 'deadlinks'

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
}

const toast = useToast()
const route = useRoute()
const router = useRouter()

const tabs: { key: TabKey; label: string }[] = [
  { key: 'io', label: '导入导出' },
  { key: 'danger', label: '清空数据' },
  { key: 'deadlinks', label: '死链检测' },
]
const tab = ref<TabKey>('io')

function applyTab() {
  const q = String(route.query.tab || '')
  if (q === 'io' || q === 'danger' || q === 'deadlinks') tab.value = q
  // legacy /admin/deadlinks redirect support via query
  if (q === 'deadlink') tab.value = 'deadlinks'
}
function setTab(key: TabKey) {
  tab.value = key
  router.replace({ query: { ...route.query, tab: key } })
}
watch(() => route.query.tab, applyTab)

const importMode = ref<'merge' | 'replace'>('merge')
const importFile = ref<File | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const importing = ref(false)
const exporting = ref(false)

function onPickFile(e: Event) {
  const input = e.target as HTMLInputElement
  importFile.value = input.files?.[0] || null
}

async function doExport() {
  exporting.value = true
  try {
    await apiDownload('/api/v1/admin/export', `booknav_export_${Date.now()}.db3`)
    toast.success('已导出数据库')
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '导出失败')
  } finally {
    exporting.value = false
  }
}

async function doImport() {
  if (!importFile.value) {
    toast.error('请选择数据库文件')
    return
  }
  if (importMode.value === 'replace' && !confirm('替换模式会清空现有分类与链接，确定？')) return
  importing.value = true
  try {
    const fd = new FormData()
    fd.append('file', importFile.value)
    fd.append('mode', importMode.value)
    const stats = await apiPostForm<{ categories: number; websites: number; skipped: number }>(
      '/api/v1/admin/import/db',
      fd,
    )
    toast.success(
      `导入完成：分类 ${stats.categories} · 链接 ${stats.websites} · 跳过 ${stats.skipped ?? 0}`,
    )
    importFile.value = null
    if (fileInput.value) fileInput.value.value = ''
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '导入失败')
  } finally {
    importing.value = false
  }
}

async function clearSites() {
  if (!confirm('确认清空全部链接？此操作不可恢复。')) return
  try {
    await apiPost('/api/v1/admin/data/clear-websites')
    toast.success('已清空全部链接')
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '操作失败')
  }
}

// —— deadlinks ——
const jobs = ref<Job[]>([])
const results = ref<DeadItem[]>([])
const batchId = ref('')
const loadingResults = ref(false)
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

async function startDeadlink() {
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
      if (j.status === 'completed' || j.status === 'failed' || j.status === 'cancelled') {
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
        toast.success(
          j.status === 'completed' ? '检测完成' : j.status === 'cancelled' ? '已停止' : '检测失败',
        )
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
  loadingResults.value = true
  try {
    results.value = await apiGet(
      `/api/v1/admin/deadlinks?batch_id=${encodeURIComponent(batchId.value)}&invalid_only=1`,
    )
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '加载结果失败')
  } finally {
    loadingResults.value = false
  }
}

onMounted(async () => {
  applyTab()
  await loadJobs()
  if (batchId.value) await loadResults()
})
</script>

<template>
  <div class="admin-page">
    <header class="page-header">
      <h1>数据管理</h1>
      <div class="page-header__actions">
        <RouterLink class="c-btn c-btn--ghost c-btn--sm" to="/admin/backups">备份管理</RouterLink>
        <RouterLink class="c-btn c-btn--ghost c-btn--sm" to="/admin/jobs">任务中心</RouterLink>
      </div>
    </header>

    <div class="c-tabs" role="tablist">
      <button
        v-for="t in tabs"
        :key="t.key"
        type="button"
        class="c-tabs__item"
        :class="{ active: tab === t.key }"
        @click="setTab(t.key)"
      >
        {{ t.label }}
      </button>
    </div>

    <section v-show="tab === 'io'" class="c-card c-card__body">
      <h3 class="c-card__title">导出</h3>
      <div class="form-row">
        <button type="button" class="c-btn c-btn--primary" :disabled="exporting" @click="doExport">
          {{ exporting ? '导出中…' : '导出数据库 (.db3)' }}
        </button>
      </div>

      <h3 class="c-card__title" style="margin-top: 22px">导入</h3>
      <div class="form-row">
        <input
          ref="fileInput"
          type="file"
          class="c-input file-input"
          accept=".db,.db3,.sqlite,.sqlite3"
          @change="onPickFile"
        />
        <select v-model="importMode" class="c-input" style="max-width: 140px">
          <option value="merge">合并</option>
          <option value="replace">替换</option>
        </select>
        <button type="button" class="c-btn c-btn--primary" :disabled="importing" @click="doImport">
          {{ importing ? '导入中…' : '开始导入' }}
        </button>
      </div>
      <p v-if="importFile" class="file-name mono">{{ importFile.name }}</p>
    </section>

    <section v-show="tab === 'danger'" class="c-card c-card__body danger-card">
      <h3 class="c-card__title">清空数据</h3>
      <div class="form-row">
        <button type="button" class="c-btn c-btn--danger" @click="clearSites">清空全部链接</button>
        <RouterLink class="c-btn c-btn--ghost c-btn--sm" to="/admin/backups">备份</RouterLink>
        <RouterLink class="c-btn c-btn--ghost c-btn--sm" to="/admin/logs">日志</RouterLink>
      </div>
    </section>

    <template v-if="tab === 'deadlinks'">
      <section class="c-card c-card__body">
        <div class="dead-head">
          <h3 class="c-card__title" style="margin-bottom: 0">死链检测</h3>
          <div class="form-row">
            <button
              type="button"
              class="c-btn c-btn--ghost c-btn--sm"
              :disabled="!batchId"
              @click="loadResults"
            >
              刷新失效列表
            </button>
            <button type="button" class="c-btn c-btn--primary" :disabled="polling" @click="startDeadlink">
              {{ polling ? '检测中…' : '开始检测' }}
            </button>
          </div>
        </div>

        <div v-if="jobs.length" class="job-list">
          <div v-for="j in jobs.slice(0, 5)" :key="j.id" class="job-line">
            <span class="mono">#{{ j.id }}</span>
            <span class="c-tag">{{ j.status }}</span>
            <span class="mono">{{ j.progress }}/{{ j.total }}</span>
            <span class="mono">ok {{ j.success }} / fail {{ j.failed }}</span>
          </div>
        </div>
      </section>

      <AdminTable
        :loading="loadingResults"
        :is-empty="!results.length"
        empty="暂无失效链接（先运行检测）"
      >
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
    </template>
  </div>
</template>

<style scoped>
.form-row {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
}
.file-input {
  max-width: 320px;
  padding: 6px 8px;
}
.file-name {
  margin: 8px 0 0;
  font-size: 12px;
  color: var(--console-text-3);
}
.mono {
  font-family: var(--console-mono, ui-monospace, monospace);
}
.danger-card {
  border-color: rgba(255, 77, 106, 0.25);
}
.dead-head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}
.job-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-top: 8px;
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
