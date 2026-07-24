<script setup lang="ts">
/**
 * 数据管理
 * - 导入导出
 * - 数据清理（按类型分项，GitHub Danger Zone 风格）
 * - 死链检测
 */
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { apiDownload, apiGet, apiPost, apiPostForm } from '@/shared/api/client'
import { useToast } from '@/shared/composables/useToast'
import AppIcon from '@/shared/ui/AppIcon.vue'
import AdminTable from '../components/AdminTable.vue'

type TabKey = 'io' | 'cleanup' | 'deadlinks'

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

interface CleanupStats {
  websites: number
  categories: number
  operation_logs: number
  jobs_finished: number
  deadlink_records: number
  icon_files: number
  icon_bytes: number
  avatar_files: number
  avatar_bytes: number
  vector_configured: boolean
  qdrant_url?: string
}

type CleanupAction =
  | 'websites'
  | 'categories'
  | 'navigation'
  | 'vectors'
  | 'icon-files'
  | 'avatar-files'
  | 'logs'
  | 'jobs'
  | 'deadlinks'

interface CleanupItem {
  id: CleanupAction
  title: string
  severity: 'warn' | 'danger'
  /** require typing this phrase to confirm (empty = simple confirm) */
  confirmPhrase?: string
  endpoint: string
  countKey?: keyof CleanupStats
  countLabel?: (n: number, s: CleanupStats) => string
  disabled?: (s: CleanupStats) => string | null
}

const toast = useToast()
const route = useRoute()
const router = useRouter()

const tabs: { key: TabKey; label: string }[] = [
  { key: 'io', label: '导入导出' },
  { key: 'cleanup', label: '数据清理' },
  { key: 'deadlinks', label: '死链检测' },
]
const tab = ref<TabKey>('io')

function applyTab() {
  const q = String(route.query.tab || '')
  if (q === 'io' || q === 'cleanup' || q === 'deadlinks') tab.value = q
  // legacy aliases
  if (q === 'danger' || q === 'clear') tab.value = 'cleanup'
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
    await loadCleanupStats()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '导入失败')
  } finally {
    importing.value = false
  }
}

// —— Cleanup (typed, no side effects) ——
const stats = ref<CleanupStats | null>(null)
const statsLoading = ref(false)
const acting = ref<CleanupAction | null>(null)

const navItems: CleanupItem[] = [
  {
    id: 'websites',
    title: '全部链接',
    severity: 'danger',
    endpoint: '/api/v1/admin/data/clear-websites',
    countKey: 'websites',
    countLabel: (n) => `${n}`,
  },
  {
    id: 'categories',
    title: '全部分类',
    severity: 'warn',
    endpoint: '/api/v1/admin/data/clear-categories',
    countKey: 'categories',
    countLabel: (n) => `${n}`,
    disabled: (s) => (s.websites > 0 ? `仍有 ${s.websites} 条链接` : null),
  },
  {
    id: 'navigation',
    title: '链接 + 分类',
    severity: 'danger',
    confirmPhrase: '清空导航',
    endpoint: '/api/v1/admin/data/clear-navigation',
    countLabel: (_n, s) => `${s.websites} / ${s.categories}`,
  },
]

const searchItems: CleanupItem[] = [
  {
    id: 'vectors',
    title: '向量索引',
    severity: 'warn',
    endpoint: '/api/v1/admin/data/clear-vectors',
    countLabel: (_n, s) => (s.vector_configured ? '已配置' : '未配置'),
    disabled: (s) => (!s.vector_configured ? '未配置 Qdrant' : null),
  },
]

const mediaItems: CleanupItem[] = [
  {
    id: 'icon-files',
    title: '站点图标',
    severity: 'warn',
    endpoint: '/api/v1/admin/data/clear-icon-files',
    countKey: 'icon_files',
    countLabel: (n, s) => `${n} · ${formatBytes(s.icon_bytes)}`,
  },
  {
    id: 'avatar-files',
    title: '用户头像',
    severity: 'warn',
    endpoint: '/api/v1/admin/data/clear-avatar-files',
    countKey: 'avatar_files',
    countLabel: (n, s) => `${n} · ${formatBytes(s.avatar_bytes)}`,
  },
]

const opsItems: CleanupItem[] = [
  {
    id: 'logs',
    title: '操作日志',
    severity: 'warn',
    endpoint: '/api/v1/admin/operation-logs/clear',
    countKey: 'operation_logs',
    countLabel: (n) => `${n}`,
  },
  {
    id: 'jobs',
    title: '已完成任务',
    severity: 'warn',
    endpoint: '/api/v1/admin/jobs/clear',
    countKey: 'jobs_finished',
    countLabel: (n) => `${n}`,
  },
  {
    id: 'deadlinks',
    title: '死链记录',
    severity: 'warn',
    endpoint: '/api/v1/admin/data/clear-deadlinks',
    countKey: 'deadlink_records',
    countLabel: (n) => `${n}`,
  },
]

const sections: { title: string; items: CleanupItem[] }[] = [
  { title: '导航数据', items: navItems },
  { title: '搜索索引', items: searchItems },
  { title: '媒体文件', items: mediaItems },
  { title: '运维记录', items: opsItems },
]

// confirm dialog state
const dialogOpen = ref(false)
const dialogItem = ref<CleanupItem | null>(null)
const dialogPhrase = ref('')
const dialogBusy = ref(false)

const dialogCanSubmit = computed(() => {
  const item = dialogItem.value
  if (!item) return false
  if (!item.confirmPhrase) return true
  return dialogPhrase.value.trim() === item.confirmPhrase
})

function formatBytes(n: number) {
  if (!n || n < 0) return '0 B'
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}

function itemCountText(item: CleanupItem): string {
  if (!stats.value) return '—'
  if (item.countLabel) {
    const n = item.countKey ? Number(stats.value[item.countKey] ?? 0) : 0
    return item.countLabel(n, stats.value)
  }
  if (item.countKey) return String(stats.value[item.countKey] ?? 0)
  return '—'
}

function itemDisabledReason(item: CleanupItem): string | null {
  if (!stats.value) return null
  return item.disabled?.(stats.value) ?? null
}

async function loadCleanupStats() {
  statsLoading.value = true
  try {
    stats.value = await apiGet<CleanupStats>('/api/v1/admin/data/cleanup-stats')
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '加载统计失败')
  } finally {
    statsLoading.value = false
  }
}

function openCleanup(item: CleanupItem) {
  const reason = itemDisabledReason(item)
  if (reason) {
    toast.error(reason)
    return
  }
  dialogItem.value = item
  dialogPhrase.value = ''
  dialogOpen.value = true
}

function closeDialog() {
  if (dialogBusy.value) return
  dialogOpen.value = false
  dialogItem.value = null
  dialogPhrase.value = ''
}

async function confirmCleanup() {
  const item = dialogItem.value
  if (!item || !dialogCanSubmit.value) return
  dialogBusy.value = true
  acting.value = item.id
  try {
    await apiPost(item.endpoint)
    toast.success(`已清理${item.title}`)
    dialogOpen.value = false
    dialogItem.value = null
    dialogPhrase.value = ''
    await loadCleanupStats()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '操作失败')
  } finally {
    dialogBusy.value = false
    acting.value = null
  }
}

// —— deadlinks ——
const jobs = ref<Job[]>([])
const results = ref<DeadItem[]>([])
const batchId = ref('')
const loadingResults = ref(false)
const polling = ref(false)

async function loadJobs() {
  const list = (await apiGet<Job[]>('/api/v1/admin/jobs')) || []
  jobs.value = list.filter((j) => j.type === 'deadlink_check')
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
  await Promise.all([loadJobs(), loadCleanupStats()])
  if (batchId.value) await loadResults()
})

watch(tab, (t) => {
  if (t === 'cleanup') loadCleanupStats()
})
</script>

<template>
  <div class="admin-page">
    <header class="page-header">
      <div>
        <h1>数据管理</h1>
      </div>
      <div class="page-header__actions">
        <RouterLink class="c-btn c-btn--ghost c-btn--sm" to="/admin/backups">备份管理</RouterLink>
        <RouterLink class="c-btn c-btn--ghost c-btn--sm" to="/admin/jobs">任务中心</RouterLink>
      </div>
    </header>

    <div class="tab-layout">
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

      <!-- 单一内容容器：保证 Tab → 面板间距在三个页签下完全一致 -->
      <section class="c-card c-card__body tab-panel">
        <!-- IO -->
        <div v-show="tab === 'io'">
          <div class="panel-head">
            <h3 class="c-card__title panel-title">导入导出</h3>
          </div>
          <h3 class="c-card__title c-card__title--sub">导出</h3>
          <div class="form-row">
            <button type="button" class="c-btn c-btn--primary" :disabled="exporting" @click="doExport">
              {{ exporting ? '导出中…' : '导出数据库 (.db3)' }}
            </button>
          </div>

          <h3 class="c-card__title c-card__title--sub" style="margin-top: 18px">导入</h3>
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
        </div>

        <!-- Cleanup -->
        <div v-show="tab === 'cleanup'">
          <div class="panel-head">
            <h3 class="c-card__title panel-title">数据清理</h3>
            <button
              type="button"
              class="c-btn c-btn--ghost c-btn--sm"
              :disabled="statsLoading"
              @click="loadCleanupStats"
            >
              {{ statsLoading ? '…' : '刷新' }}
            </button>
          </div>

          <div class="cleanup">
            <div v-for="sec in sections" :key="sec.title" class="dz-module">
              <div class="dz-module__label">{{ sec.title }}</div>
              <div class="dz-chips">
                <div
                  v-for="item in sec.items"
                  :key="item.id"
                  class="dz-chip"
                  :class="{
                    'dz-chip--danger': item.severity === 'danger',
                    'dz-chip--disabled': !!itemDisabledReason(item),
                  }"
                  :title="itemDisabledReason(item) || item.title"
                >
                  <span class="dz-chip__name">{{ item.title }}</span>
                  <span class="dz-chip__count">{{ itemCountText(item) }}</span>
                  <button
                    type="button"
                    class="dz-chip__btn"
                    :disabled="!!itemDisabledReason(item) || acting === item.id"
                    :aria-label="`清理${item.title}`"
                    @click="openCleanup(item)"
                  >
                    <AppIcon name="trash-2" :size="15" />
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Deadlinks -->
        <div v-show="tab === 'deadlinks'">
          <div class="panel-head">
            <h3 class="c-card__title panel-title">死链检测</h3>
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

          <div class="dead-table-wrap">
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
          </div>
        </div>
      </section>
    </div>

    <!-- Confirm modal -->
    <Teleport to="body">
      <div v-if="dialogOpen && dialogItem" class="dz-modal-root" role="dialog" aria-modal="true">
        <div class="dz-modal-backdrop" @click="closeDialog" />
        <div class="dz-modal">
          <h3 class="dz-modal__title">清理{{ dialogItem.title }}</h3>
          <p class="dz-modal__count">{{ itemCountText(dialogItem) }}</p>

          <div v-if="dialogItem.confirmPhrase" class="dz-modal__phrase">
            <label>
              输入
              <code>{{ dialogItem.confirmPhrase }}</code>
              <input
                v-model="dialogPhrase"
                class="c-input"
                type="text"
                autocomplete="off"
                :placeholder="dialogItem.confirmPhrase"
                @keyup.enter="confirmCleanup"
              />
            </label>
          </div>
          <p v-else class="dz-modal__warn">不可恢复，确认清理？</p>

          <div class="dz-modal__actions">
            <button type="button" class="c-btn c-btn--ghost" :disabled="dialogBusy" @click="closeDialog">
              取消
            </button>
            <button
              type="button"
              class="c-btn c-btn--danger"
              :disabled="!dialogCanSubmit || dialogBusy"
              @click="confirmCleanup"
            >
              {{ dialogBusy ? '…' : '确认' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
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

/* Tab → 单一面板：间距固定，不随页签变化 */
.tab-layout {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
  min-width: 0;
}
.tab-panel {
  margin: 0;
}

/* Shared panel header (aligned across tabs) */
.panel-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin: 0 0 14px;
  min-height: 32px;
}
.panel-title {
  margin: 0 !important;
}
.c-card__title--sub {
  margin: 0 0 10px;
  font-size: 13px;
  font-weight: 600;
  color: var(--console-text-2);
}

/* —— Cleanup: compact adaptive chips inside card —— */
.cleanup {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.dz-module {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.dz-module__label {
  font-size: 12px;
  font-weight: 600;
  color: var(--console-text-3);
  letter-spacing: 0.02em;
  padding-left: 2px;
}

/* chips wrap — adaptive columns via flex-wrap */
.dz-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.dz-chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 36px;
  padding: 4px 4px 4px 12px;
  border-radius: 10px;
  border: 1px solid var(--console-border);
  background: var(--console-surface);
  transition: border-color 0.15s, background 0.15s;
}
.dz-chip:hover:not(.dz-chip--disabled) {
  border-color: rgba(148, 163, 184, 0.35);
  background: rgba(255, 255, 255, 0.02);
}
.dz-chip--danger {
  border-color: rgba(239, 68, 68, 0.22);
}
.dz-chip--disabled {
  opacity: 0.55;
}
.dz-chip__name {
  font-size: 13px;
  font-weight: 500;
  color: var(--console-text);
  white-space: nowrap;
}
.dz-chip__count {
  font-size: 11px;
  font-family: var(--console-mono, ui-monospace, monospace);
  color: var(--console-text-3);
  padding: 2px 7px;
  border-radius: 999px;
  background: rgba(0, 0, 0, 0.25);
  border: 1px solid var(--console-border);
  white-space: nowrap;
}
.dz-chip__btn {
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  margin: 0;
  padding: 0;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--console-text-3);
  cursor: pointer;
  transition: color 0.15s, background 0.15s;
}
.dz-chip__btn:hover:not(:disabled) {
  color: #fca5a5;
  background: rgba(239, 68, 68, 0.12);
}
.dz-chip__btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.dz-chip--danger .dz-chip__btn {
  color: #f87171;
}

.job-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0 0 14px;
}
.dead-table-wrap {
  margin-top: 4px;
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

/* Modal */
.dz-modal-root {
  position: fixed;
  inset: 0;
  z-index: 2000;
  display: grid;
  place-items: center;
  padding: 16px;
}
.dz-modal-backdrop {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  backdrop-filter: blur(2px);
}
.dz-modal {
  position: relative;
  width: min(360px, 100%);
  padding: 18px;
  border-radius: var(--console-radius);
  border: 1px solid var(--console-border);
  background: #12171f;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.45);
}
.dz-modal__title {
  margin: 0 0 6px;
  font-size: 15px;
  font-weight: 650;
}
.dz-modal__count {
  margin: 0 0 12px;
  font-family: var(--console-mono, ui-monospace, monospace);
  font-size: 12px;
  color: var(--console-text-3);
}
.dz-modal__phrase label {
  display: grid;
  gap: 8px;
  font-size: 13px;
  color: var(--console-text-2);
}
.dz-modal__phrase code {
  color: #fca5a5;
  font-size: 12px;
  padding: 1px 6px;
  border-radius: 4px;
  background: rgba(239, 68, 68, 0.12);
}
.dz-modal__warn {
  margin: 0 0 8px;
  font-size: 13px;
  color: var(--console-text-2);
}
.dz-modal__actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 14px;
}
</style>
