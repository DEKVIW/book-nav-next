<script setup lang="ts">
/**
 * 任务中心：进度监控 / 停止 / 删除
 * 3s 轮询足够展示进度，无需 SSE/WebSocket。
 * 轮询必须 quiet：禁止整表 loading，失败保留旧数据。
 */
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { apiDelete, apiGet, apiPost } from '@/shared/api/client'
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
const jobs = ref<Job[]>([])
/** true only while a non-quiet request is in flight */
const loading = ref(false)
const loadError = ref('')
const acting = ref<number | null>(null)
let timer: ReturnType<typeof setInterval> | null = null
/** monotonic request id — drop stale responses */
let loadSeq = 0
/** count of non-quiet loads so concurrent refresh cannot stick loading forever */
let nonQuietInflight = 0

const typeLabel: Record<string, string> = {
  deadlink_check: '死链检测',
  icon_sync: '图标抓取',
  vector_index: '向量索引',
}

const typeLink: Record<string, string> = {
  deadlink_check: '/admin/data?tab=deadlinks',
  icon_sync: '/admin/icons',
  vector_index: '/admin/settings?tab=vector',
}

const hasActive = computed(() => jobs.value.some((j) => isActive(j)))

/** API may return null (Go nil slice), {items:[]}, or Job[] */
function normalizeList(data: unknown): Job[] {
  if (data == null) return []
  if (Array.isArray(data)) return data as Job[]
  if (typeof data === 'object') {
    const o = data as { items?: unknown; data?: unknown; list?: unknown }
    if (Array.isArray(o.items)) return o.items as Job[]
    if (Array.isArray(o.list)) return o.list as Job[]
    if (Array.isArray(o.data)) return o.data as Job[]
  }
  return []
}

async function load(opts?: { quiet?: boolean }) {
  const quiet = !!opts?.quiet
  const seq = ++loadSeq
  if (!quiet) {
    nonQuietInflight++
    loading.value = true
    loadError.value = ''
  }
  try {
    const data = await apiGet<Job[] | null>('/api/v1/admin/jobs')
    if (seq !== loadSeq) return
    // Always assign a real array — never null/undefined (breaks is-empty / v-for)
    jobs.value = normalizeList(data)
    loadError.value = ''
  } catch (e: unknown) {
    if (seq !== loadSeq) return
    // quiet poll: keep previous rows, no toast
    if (!quiet) {
      loadError.value = e instanceof Error ? e.message : '加载失败'
      toast.error(loadError.value)
    }
  } finally {
    if (!quiet) {
      nonQuietInflight = Math.max(0, nonQuietInflight - 1)
      if (nonQuietInflight === 0) loading.value = false
    }
  }
}

async function stop(id: number) {
  if (!confirm(`停止任务 #${id}？进行中的项会在当前批次后退出。`)) return
  acting.value = id
  try {
    await apiPost(`/api/v1/admin/jobs/${id}/cancel`)
    toast.success('已请求停止')
    await load({ quiet: true })
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '停止失败')
  } finally {
    acting.value = null
  }
}

async function remove(id: number) {
  if (!confirm(`删除任务 #${id}？`)) return
  acting.value = id
  try {
    await apiDelete(`/api/v1/admin/jobs/${id}`)
    toast.success('已删除')
    await load({ quiet: true })
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '删除失败')
  } finally {
    acting.value = null
  }
}

async function clearFinished() {
  if (!confirm('清空所有已完成 / 失败 / 已停止的任务？')) return
  try {
    const r = await apiPost<{ deleted: number }>('/api/v1/admin/jobs/clear')
    toast.success(`已清理 ${r.deleted ?? 0} 条`)
    await load({ quiet: true })
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '清理失败')
  }
}

function isActive(j: Job) {
  return j.status === 'pending' || j.status === 'running'
}

function statusClass(status: string) {
  if (status === 'running' || status === 'pending') return 'c-tag c-tag--run'
  if (status === 'completed') return 'c-tag c-tag--ok'
  if (status === 'failed') return 'c-tag c-tag--fail'
  if (status === 'cancelled') return 'c-tag c-tag--stop'
  return 'c-tag'
}

function pollTick() {
  if (typeof document !== 'undefined' && document.hidden) return
  void load({ quiet: true })
}

onMounted(() => {
  void load()
  timer = setInterval(pollTick, 3000)
  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', onVisible)
  }
})
function onVisible() {
  if (!document.hidden) void load({ quiet: true })
}
onUnmounted(() => {
  if (timer) clearInterval(timer)
  timer = null
  loadSeq++ // invalidate in-flight
  if (typeof document !== 'undefined') {
    document.removeEventListener('visibilitychange', onVisible)
  }
})
</script>

<template>
  <div class="admin-page">
    <header class="page-header">
      <h1>任务中心</h1>
      <div class="page-header__actions">
        <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="clearFinished">清空已结束</button>
        <button type="button" class="c-btn c-btn--ghost c-btn--sm" :disabled="loading" @click="load()">
          刷新
        </button>
      </div>
    </header>

    <section class="c-card c-card__body launch-hint">
      <div class="launch-grid">
        <RouterLink class="launch-item" to="/admin/settings?tab=vector">向量索引</RouterLink>
        <RouterLink class="launch-item" to="/admin/icons">图标抓取</RouterLink>
        <RouterLink class="launch-item" to="/admin/data?tab=deadlinks">死链检测</RouterLink>
      </div>
    </section>

    <p v-if="loadError && !jobs.length" class="jobs-error">{{ loadError }}</p>
    <p v-if="hasActive" class="jobs-live">有进行中的任务 · 每 3 秒自动刷新</p>

    <AdminTable :loading="loading" :is-empty="jobs.length === 0" empty="暂无任务记录">
      <template #head>
        <tr>
          <th class="c-col-id">ID</th>
          <th>类型</th>
          <th style="width: 110px">状态</th>
          <th style="width: 120px">进度</th>
          <th style="width: 72px">成功</th>
          <th style="width: 72px">失败</th>
          <th>错误</th>
          <th class="c-col-actions">操作</th>
        </tr>
      </template>
      <tr v-for="j in jobs" :key="j.id">
        <td>{{ j.id }}</td>
        <td>
          <RouterLink v-if="typeLink[j.type]" class="type-link" :to="typeLink[j.type]">
            {{ typeLabel[j.type] || j.type }}
          </RouterLink>
          <span v-else>{{ typeLabel[j.type] || j.type }}</span>
        </td>
        <td><span :class="statusClass(j.status)">{{ j.status }}</span></td>
        <td class="mono">{{ j.progress }}/{{ j.total }}</td>
        <td class="mono">{{ j.success }}</td>
        <td class="mono">{{ j.failed }}</td>
        <td>
          <div class="c-cell-ellipsis" :title="j.error">{{ j.error || '—' }}</div>
        </td>
        <td class="c-col-actions">
          <div class="c-cell-actions">
            <button
              v-if="isActive(j)"
              type="button"
              class="c-btn c-btn--ghost c-btn--sm"
              :disabled="acting === j.id"
              @click="stop(j.id)"
            >
              停止
            </button>
            <button
              v-else
              type="button"
              class="c-btn c-btn--ghost c-btn--sm"
              :disabled="acting === j.id"
              @click="remove(j.id)"
            >
              删除
            </button>
          </div>
        </td>
      </tr>
    </AdminTable>
  </div>
</template>

<style scoped>
.launch-hint {
  margin: 0;
}
.launch-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}
.launch-item {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 12px 14px;
  border-radius: var(--console-radius, 8px);
  border: 1px solid var(--console-border);
  background: rgba(0, 0, 0, 0.15);
  text-decoration: none;
  color: inherit;
  font-size: 13px;
  font-weight: 600;
  transition: border-color 0.12s, background 0.12s;
}
.launch-item:hover {
  border-color: rgba(94, 240, 255, 0.35);
  background: rgba(61, 231, 255, 0.06);
}
.type-link {
  color: var(--energy, #3de7ff);
  text-decoration: none;
}
.type-link:hover {
  text-decoration: underline;
}
.mono {
  font-family: var(--font-mono, ui-monospace, monospace);
  font-size: 12px;
}
.jobs-error {
  margin: 0;
  font-size: 13px;
  color: #fca5a5;
}
.jobs-live {
  margin: 0;
  font-size: 12px;
  color: var(--console-text-3);
}
:deep(.c-tag--run) {
  color: #5ef0ff;
  border-color: rgba(94, 240, 255, 0.35);
  background: rgba(94, 240, 255, 0.1);
}
:deep(.c-tag--ok) {
  color: #3dffb5;
  border-color: rgba(61, 255, 181, 0.35);
  background: rgba(61, 255, 181, 0.1);
}
:deep(.c-tag--fail) {
  color: #ff6b8a;
  border-color: rgba(255, 107, 138, 0.35);
  background: rgba(255, 107, 138, 0.1);
}
:deep(.c-tag--stop) {
  color: #c5d0e0;
  border-color: rgba(180, 190, 210, 0.3);
  background: rgba(180, 190, 210, 0.08);
}
@media (max-width: 900px) {
  .launch-grid {
    grid-template-columns: 1fr;
  }
}
</style>
