<script setup lang="ts">
/**
 * 数据管理（Tab）
 * - 导入导出
 * - 清空数据
 * - 死链检测（业务启动 + 结果，任务中心只看进度）
 */
import { onMounted, ref, watch } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { apiGet, apiPost } from '@/shared/api/client'
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

const legacyFile = ref('')
const legacyMode = ref<'merge' | 'replace'>('merge')
const importing = ref(false)
const exporting = ref(false)

async function doExport() {
  exporting.value = true
  try {
    const data = await apiGet('/api/v1/admin/export')
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = `booknav-export-${Date.now()}.json`
    a.click()
    URL.revokeObjectURL(a.href)
    toast.success('已导出 JSON')
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '导出失败')
  } finally {
    exporting.value = false
  }
}

async function doLegacyImport() {
  if (!legacyFile.value.trim()) {
    toast.error('请填写 data 目录下的文件名')
    return
  }
  if (legacyMode.value === 'replace' && !confirm('替换模式会清空现有分类与链接，确定？')) return
  importing.value = true
  try {
    const stats = await apiPost<{ categories: number; websites: number; skipped: number }>(
      '/api/v1/admin/import/legacy-db3',
      { filename: legacyFile.value.trim(), mode: legacyMode.value },
    )
    toast.success(
      `导入完成：分类 ${stats.categories} · 链接 ${stats.websites} · 跳过 ${stats.skipped ?? 0}`,
    )
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
      <div>
        <h1>数据管理</h1>
        <p>导入导出 · 清空 · 死链检测</p>
      </div>
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
      <h3 class="c-card__title">旧站导入（Flask .db3 / app.db）</h3>
      <p class="field-hint">
        文件放到服务器 <code>data/</code>（容器 <code>/data/</code>）后填写文件名。合并模式更安全。
      </p>
      <div class="form-row">
        <input
          v-model="legacyFile"
          class="c-input"
          placeholder="例如 booknav_export.db3"
          style="max-width: 320px"
        />
        <select v-model="legacyMode" class="c-input" style="max-width: 140px">
          <option value="merge">合并（推荐）</option>
          <option value="replace">替换</option>
        </select>
        <button type="button" class="c-btn c-btn--primary" :disabled="importing" @click="doLegacyImport">
          {{ importing ? '导入中…' : '开始导入' }}
        </button>
      </div>

      <h3 class="c-card__title" style="margin-top: 22px">导出</h3>
      <p class="field-hint">导出分类与链接 JSON（不是完整库备份，库备份请用备份管理）。</p>
      <button type="button" class="c-btn c-btn--ghost" :disabled="exporting" @click="doExport">
        {{ exporting ? '导出中…' : '导出 JSON' }}
      </button>
    </section>

    <section v-show="tab === 'danger'" class="c-card c-card__body danger-card">
      <h3 class="c-card__title">清空数据</h3>
      <p class="field-hint">清空后无法通过本页恢复，请先到「备份管理」做快照。</p>
      <div class="form-row">
        <button type="button" class="c-btn c-btn--danger" @click="clearSites">清空全部链接</button>
        <RouterLink class="c-btn c-btn--ghost c-btn--sm" to="/admin/backups">去备份</RouterLink>
        <RouterLink class="c-btn c-btn--ghost c-btn--sm" to="/admin/logs">操作日志</RouterLink>
      </div>
    </section>

    <template v-if="tab === 'deadlinks'">
      <section class="c-card c-card__body">
        <div class="dead-head">
          <div>
            <h3 class="c-card__title" style="margin-bottom: 4px">死链检测</h3>
            <p class="field-hint" style="margin: 0">批量检测可访问性；进度也可在任务中心查看与停止。</p>
          </div>
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
.field-hint {
  margin: 0 0 14px;
  font-size: 12px;
  color: var(--console-text-3);
  line-height: 1.5;
}
.field-hint code {
  font-family: var(--font-mono, ui-monospace, monospace);
  font-size: 11px;
  padding: 1px 5px;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.25);
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
