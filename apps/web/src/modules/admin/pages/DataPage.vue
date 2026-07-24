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
  desc: string
  keeps: string
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
    title: '清空全部链接',
    desc: '删除所有网站链接记录。',
    keeps: '不影响分类、向量索引、图标文件、日志。',
    severity: 'danger',
    endpoint: '/api/v1/admin/data/clear-websites',
    countKey: 'websites',
    countLabel: (n) => `${n} 条链接`,
  },
  {
    id: 'categories',
    title: '清空全部分类',
    desc: '删除全部分类结构（含父子分类）。',
    keeps: '不影响链接（若仍有链接将拒绝执行）。',
    severity: 'warn',
    endpoint: '/api/v1/admin/data/clear-categories',
    countKey: 'categories',
    countLabel: (n) => `${n} 个分类`,
    disabled: (s) =>
      s.websites > 0 ? `仍有 ${s.websites} 条链接，请先清空链接` : null,
  },
  {
    id: 'navigation',
    title: '清空导航数据',
    desc: '同时删除全部链接与全部分类，使导航归零。',
    keeps: '不影响向量、图标文件、用户与站点设置。',
    severity: 'danger',
    confirmPhrase: '清空导航',
    endpoint: '/api/v1/admin/data/clear-navigation',
    countLabel: (_n, s) => `${s.websites} 链接 · ${s.categories} 分类`,
  },
]

const searchItems: CleanupItem[] = [
  {
    id: 'vectors',
    title: '清空向量索引',
    desc: '删除 Qdrant 中的网站向量集合（用于 AI / 语义搜索）。',
    keeps: '不影响数据库中的链接与分类；之后可重新建索引。',
    severity: 'warn',
    endpoint: '/api/v1/admin/data/clear-vectors',
    countLabel: (_n, s) => (s.vector_configured ? '已配置' : '未配置'),
    disabled: (s) => (!s.vector_configured ? '未配置 Qdrant，无需清理' : null),
  },
]

const mediaItems: CleanupItem[] = [
  {
    id: 'icon-files',
    title: '清空站点图标文件',
    desc: '删除 data/uploads/icons 下已下载的图标文件。',
    keeps: '不修改链接的 icon 字段；前台可能出现图标失效直至重新抓取。',
    severity: 'warn',
    endpoint: '/api/v1/admin/data/clear-icon-files',
    countKey: 'icon_files',
    countLabel: (n, s) => `${n} 个文件 · ${formatBytes(s.icon_bytes)}`,
  },
  {
    id: 'avatar-files',
    title: '清空用户头像文件',
    desc: '删除 data/uploads/avatars 下的头像文件。',
    keeps: '不修改用户 avatar 字段。',
    severity: 'warn',
    endpoint: '/api/v1/admin/data/clear-avatar-files',
    countKey: 'avatar_files',
    countLabel: (n, s) => `${n} 个文件 · ${formatBytes(s.avatar_bytes)}`,
  },
]

const opsItems: CleanupItem[] = [
  {
    id: 'logs',
    title: '清空操作日志',
    desc: '删除全部操作审计记录。',
    keeps: '不影响链接与分类数据。',
    severity: 'warn',
    endpoint: '/api/v1/admin/operation-logs/clear',
    countKey: 'operation_logs',
    countLabel: (n) => `${n} 条`,
  },
  {
    id: 'jobs',
    title: '清空已完成任务',
    desc: '删除状态为已完成 / 失败 / 已取消的任务记录。',
    keeps: '进行中的任务不会被删除。',
    severity: 'warn',
    endpoint: '/api/v1/admin/jobs/clear',
    countKey: 'jobs_finished',
    countLabel: (n) => `${n} 条`,
  },
  {
    id: 'deadlinks',
    title: '清空死链检测记录',
    desc: '删除历史死链检测结果。',
    keeps: '不影响链接本身与 is_valid 标记。',
    severity: 'warn',
    endpoint: '/api/v1/admin/data/clear-deadlinks',
    countKey: 'deadlink_records',
    countLabel: (n) => `${n} 条`,
  },
]

const sections: { title: string; hint: string; items: CleanupItem[] }[] = [
  {
    title: '导航数据',
    hint: '前台分类与链接。每项只删除自身范围。',
    items: navItems,
  },
  {
    title: '搜索索引',
    hint: '外部向量库，与 SQLite 相互独立。',
    items: searchItems,
  },
  {
    title: '媒体文件',
    hint: '仅删除磁盘文件，不改数据库字段。',
    items: mediaItems,
  },
  {
    title: '运维记录',
    hint: '日志与任务历史，不影响业务内容。',
    items: opsItems,
  },
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
    toast.success(item.title.replace(/^清空/, '已清空') || '清理完成')
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

    <!-- IO -->
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

    <!-- Cleanup: GitHub-style Danger Zone -->
    <section v-show="tab === 'cleanup'" class="cleanup">
      <div class="cleanup-banner">
        <div class="cleanup-banner__text">
          <strong>按类型清理</strong>
          <span>每一项只删除自身数据，不会顺带清理其它类型。操作不可恢复，建议先备份。</span>
        </div>
        <button
          type="button"
          class="c-btn c-btn--ghost c-btn--sm"
          :disabled="statsLoading"
          @click="loadCleanupStats"
        >
          {{ statsLoading ? '刷新中…' : '刷新统计' }}
        </button>
      </div>

      <div class="cleanup-grid">
        <div v-for="sec in sections" :key="sec.title" class="dz-section">
          <div class="dz-section__head">
            <h3 class="dz-section__title">{{ sec.title }}</h3>
            <p class="dz-section__hint">{{ sec.hint }}</p>
          </div>
          <ul class="dz-list">
            <li v-for="item in sec.items" :key="item.id" class="dz-row">
              <div class="dz-row__body">
                <div class="dz-row__title-line">
                  <span class="dz-row__title">{{ item.title }}</span>
                  <span class="dz-count">{{ itemCountText(item) }}</span>
                </div>
                <p class="dz-row__desc">{{ item.desc }}</p>
                <p class="dz-row__keeps">{{ item.keeps }}</p>
                <p v-if="itemDisabledReason(item)" class="dz-row__block">
                  {{ itemDisabledReason(item) }}
                </p>
              </div>
              <div class="dz-row__action">
                <button
                  type="button"
                  class="c-btn c-btn--sm"
                  :class="item.severity === 'danger' ? 'c-btn--danger' : 'c-btn--ghost'"
                  :disabled="!!itemDisabledReason(item) || acting === item.id"
                  @click="openCleanup(item)"
                >
                  {{ acting === item.id ? '处理中…' : '清理' }}
                </button>
              </div>
            </li>
          </ul>
        </div>
      </div>
    </section>

    <!-- Deadlinks -->
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

    <!-- Confirm modal -->
    <Teleport to="body">
      <div v-if="dialogOpen && dialogItem" class="dz-modal-root" role="dialog" aria-modal="true">
        <div class="dz-modal-backdrop" @click="closeDialog" />
        <div class="dz-modal">
          <h3 class="dz-modal__title">{{ dialogItem.title }}</h3>
          <p class="dz-modal__desc">{{ dialogItem.desc }}</p>
          <p class="dz-modal__keeps">{{ dialogItem.keeps }}</p>
          <p class="dz-modal__count">当前：{{ itemCountText(dialogItem) }}</p>

          <div v-if="dialogItem.confirmPhrase" class="dz-modal__phrase">
            <label>
              请输入
              <code>{{ dialogItem.confirmPhrase }}</code>
              以确认
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
          <p v-else class="dz-modal__warn">此操作不可恢复，确定继续？</p>

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
              {{ dialogBusy ? '处理中…' : '确认清理' }}
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

/* —— Cleanup / Danger Zone (GitHub-like) —— */
.cleanup {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.cleanup-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
  align-items: start;
}
@media (max-width: 960px) {
  .cleanup-grid {
    grid-template-columns: 1fr;
  }
}
.cleanup-banner {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border-radius: var(--console-radius);
  border: 1px solid rgba(239, 68, 68, 0.22);
  background: rgba(239, 68, 68, 0.06);
}
.cleanup-banner__text {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 13px;
  color: var(--console-text-2);
  line-height: 1.45;
  max-width: 640px;
}
.cleanup-banner__text strong {
  color: #fca5a5;
  font-size: 13px;
  font-weight: 600;
}

.dz-section {
  border: 1px solid var(--console-border);
  border-radius: var(--console-radius);
  background: var(--console-surface);
  overflow: hidden;
}
.dz-section__head {
  padding: 14px 16px 10px;
  border-bottom: 1px solid var(--console-border);
}
.dz-section__title {
  margin: 0;
  font-size: 14px;
  font-weight: 650;
  color: var(--console-text);
}
.dz-section__hint {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--console-text-3);
  line-height: 1.4;
}

.dz-list {
  list-style: none;
  margin: 0;
  padding: 0;
}
.dz-row {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px 12px;
  padding: 12px 14px;
  border-bottom: 1px solid var(--console-border);
}
.dz-row:last-child {
  border-bottom: 0;
}
.dz-row__body {
  flex: 1 1 160px;
  min-width: 0;
}
.dz-row__title-line {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}
.dz-row__title {
  font-size: 13px;
  font-weight: 600;
  color: var(--console-text);
}
.dz-count {
  font-size: 11px;
  font-family: var(--console-mono, ui-monospace, monospace);
  color: var(--console-text-3);
  padding: 2px 8px;
  border-radius: 999px;
  border: 1px solid var(--console-border);
  background: rgba(0, 0, 0, 0.2);
}
.dz-row__desc {
  margin: 0 0 4px;
  font-size: 12px;
  color: var(--console-text-2);
  line-height: 1.45;
}
.dz-row__keeps {
  margin: 0;
  font-size: 11px;
  color: var(--console-text-3);
  line-height: 1.4;
}
.dz-row__block {
  margin: 6px 0 0;
  font-size: 12px;
  color: #fbbf24;
}
.dz-row__action {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
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
  width: min(420px, 100%);
  padding: 20px;
  border-radius: var(--console-radius);
  border: 1px solid var(--console-border);
  background: #12171f;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.45);
}
.dz-modal__title {
  margin: 0 0 8px;
  font-size: 16px;
  font-weight: 650;
}
.dz-modal__desc,
.dz-modal__keeps,
.dz-modal__count,
.dz-modal__warn {
  margin: 0 0 8px;
  font-size: 13px;
  color: var(--console-text-2);
  line-height: 1.45;
}
.dz-modal__keeps {
  font-size: 12px;
  color: var(--console-text-3);
}
.dz-modal__count {
  font-family: var(--console-mono, ui-monospace, monospace);
  font-size: 12px;
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
  color: #fbbf24;
}
.dz-modal__actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}
</style>
