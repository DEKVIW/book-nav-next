<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { apiDelete, apiGet, apiPost } from '@/shared/api/client'
import { useToast } from '@/shared/composables/useToast'
import AdminTable from '../components/AdminTable.vue'

interface Backup {
  name: string
  size?: number
  mod_time?: string
  kind?: string
}

interface WebDAVConfig {
  id: number
  name: string
  webdav_url: string
  webdav_username: string
  webdav_password?: string
  password_configured?: boolean
  webdav_path: string
  enabled: boolean
  auto_backup: boolean
  backup_data: boolean
  backup_config: boolean
  backup_interval: number
  backup_keep_count: number
  last_backup_time?: string
  last_backup_status?: string
}

const toast = useToast()
const tab = ref<'local' | 'cloud'>('local')
const items = ref<Backup[]>([])
const configs = ref<WebDAVConfig[]>([])
const loading = ref(false)
const cloudLoading = ref(false)
const runningCloud = ref<number | null>(null)
const showEditor = ref(false)
const editor = reactive({
  id: 0,
  name: '',
  webdav_url: '',
  webdav_username: '',
  webdav_password: '',
  webdav_path: '/nav_backups/',
  enabled: true,
  auto_backup: false,
  backup_data: true,
  backup_config: true,
  backup_interval: 24,
  backup_keep_count: 10,
})

async function loadLocal() {
  loading.value = true
  try {
    items.value = (await apiGet('/api/v1/admin/backups')) || []
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

async function loadCloud() {
  cloudLoading.value = true
  try {
    configs.value = (await apiGet('/api/v1/admin/webdav')) || []
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    cloudLoading.value = false
  }
}

async function createData() {
  try {
    const r = await apiPost<{ name: string }>('/api/v1/admin/backups', { kind: 'data' })
    toast.success(`数据备份 ${r.name}`)
    await loadLocal()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '备份失败')
  }
}

async function createConfig() {
  try {
    // Dedicated path (not /backups/config — that is swallowed by {name})
    const r = await apiPost<{ name: string }>('/api/v1/admin/backups/create-config')
    toast.success(`配置备份 ${r.name}`)
    await loadLocal()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '备份失败')
  }
}

async function remove(name: string) {
  if (!confirm(`删除备份 ${name}？`)) return
  await apiDelete(`/api/v1/admin/backups/${encodeURIComponent(name)}`)
  toast.success('已删除')
  await loadLocal()
}

function kindLabel(b: Backup) {
  if (b.kind === 'config' || isConfigName(b.name)) return '配置'
  if (b.kind === 'data') return '数据'
  if (b.kind === 'legacy-db') return '旧库'
  return '—'
}

function isConfigName(name: string) {
  const n = name.toLowerCase()
  return n.endsWith('.config.json') || (n.includes('config') && n.endsWith('.json'))
}

async function restore(b: Backup) {
  const isConfig = b.kind === 'config' || isConfigName(b.name)
  const msg = isConfig
    ? `恢复配置备份 ${b.name}？将覆盖站点/AI/图标等设置。`
    : `恢复数据备份 ${b.name}？将覆盖数据库与本地图标文件。`
  if (!confirm(msg)) return
  try {
    await apiPost(`/api/v1/admin/backups/${encodeURIComponent(b.name)}/restore`)
    toast.success(isConfig ? '配置已恢复' : '数据已恢复，建议重启服务')
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '恢复失败')
  }
}

function openNew() {
  Object.assign(editor, {
    id: 0,
    name: '',
    webdav_url: '',
    webdav_username: '',
    webdav_password: '',
    webdav_path: '/nav_backups/',
    enabled: true,
    auto_backup: false,
    backup_data: true,
    backup_config: true,
    backup_interval: 24,
    backup_keep_count: 10,
  })
  showEditor.value = true
}

function openEdit(c: WebDAVConfig) {
  Object.assign(editor, {
    id: c.id,
    name: c.name,
    webdav_url: c.webdav_url,
    webdav_username: c.webdav_username,
    webdav_password: '',
    webdav_path: c.webdav_path || '/nav_backups/',
    enabled: c.enabled,
    auto_backup: c.auto_backup,
    backup_data: c.backup_data !== false,
    backup_config: c.backup_config !== false,
    backup_interval: c.backup_interval || 24,
    backup_keep_count: c.backup_keep_count || 10,
  })
  showEditor.value = true
}

async function saveConfig() {
  if (!editor.backup_data && !editor.backup_config) {
    toast.error('请至少勾选数据备份或配置备份')
    return
  }
  try {
    const body: Record<string, unknown> = {
      id: editor.id || undefined,
      name: editor.name,
      webdav_url: editor.webdav_url,
      webdav_username: editor.webdav_username,
      webdav_path: editor.webdav_path,
      enabled: editor.enabled,
      auto_backup: editor.auto_backup,
      backup_data: editor.backup_data,
      backup_config: editor.backup_config,
      backup_interval: editor.backup_interval,
      backup_keep_count: editor.backup_keep_count,
    }
    if (editor.webdav_password && !editor.webdav_password.startsWith('****')) {
      body.webdav_password = editor.webdav_password
    }
    configs.value = await apiPost('/api/v1/admin/webdav', body)
    toast.success('已保存')
    showEditor.value = false
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '保存失败')
  }
}

function kindsText(c: WebDAVConfig) {
  const parts: string[] = []
  if (c.backup_data !== false) parts.push('数据')
  if (c.backup_config !== false) parts.push('配置')
  return parts.length ? parts.join(' · ') : '—'
}

async function runCloudBackup(id: number) {
  runningCloud.value = id
  try {
    const r = await apiPost<{ uploaded?: string[] }>(`/api/v1/admin/webdav/${id}/run-backup`, {})
    const n = r?.uploaded?.length || 0
    toast.success(n ? `已上传 ${n} 个备份` : '已上传')
    await loadCloud()
    await loadLocal()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '云端备份失败')
  } finally {
    runningCloud.value = null
  }
}

async function deleteConfig(id: number) {
  if (!confirm('删除该配置？')) return
  try {
    configs.value = await apiDelete(`/api/v1/admin/webdav/${id}`)
    toast.success('已删除')
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '删除失败')
  }
}

async function testConfig(id: number) {
  try {
    const r = await apiPost<{ ok: boolean; message: string }>(`/api/v1/admin/webdav/${id}/test`)
    if (r.ok) toast.success(r.message)
    else toast.error(r.message)
    await loadCloud()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '连接失败')
  }
}

async function uploadToCloud(filename: string, configId: number) {
  try {
    await apiPost(`/api/v1/admin/webdav/${configId}/upload`, { filename })
    toast.success('已上传')
    await loadCloud()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '上传失败')
  }
}

function fmtSize(n?: number) {
  if (!n) return '—'
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(2)} MB`
}

function statusText(s?: string) {
  if (!s) return '—'
  const [st, msg] = s.split('|', 2)
  return `${st === 'success' ? '成功' : '失败'}${msg ? ' · ' + msg : ''}`
}

onMounted(() => {
  loadLocal()
  loadCloud()
})
</script>

<template>
  <div class="admin-page">
    <header class="page-header">
      <h1>备份管理</h1>
    </header>

    <div class="settings-layout">
      <div class="c-tabs" role="tablist">
        <button type="button" class="c-tabs__item" :class="{ active: tab === 'local' }" @click="tab = 'local'">
          本地备份
        </button>
        <button type="button" class="c-tabs__item" :class="{ active: tab === 'cloud' }" @click="tab = 'cloud'">
          云端备份
        </button>
      </div>

      <div class="settings-panel">
        <!-- 本地：结构与其它模块一致 -->
        <section v-show="tab === 'local'" class="panel-card">
          <div class="panel-toolbar">
            <span class="panel-meta">共 {{ items.length }} 个</span>
            <div class="actions-row">
              <button type="button" class="c-btn c-btn--primary c-btn--sm" @click="createData">
                数据备份
              </button>
              <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="createConfig">
                配置备份
              </button>
              <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="loadLocal">刷新</button>
            </div>
          </div>

          <AdminTable :loading="loading" :is-empty="!items.length" empty="暂无本地备份">
            <template #head>
              <tr>
                <th style="width: 72px">类型</th>
                <th>文件名</th>
                <th style="width: 100px">大小</th>
                <th class="c-col-date">时间</th>
                <th class="c-col-actions">操作</th>
              </tr>
            </template>
            <tr v-for="b in items" :key="b.name">
              <td><span class="c-tag">{{ kindLabel(b) }}</span></td>
              <td>
                <div class="c-cell-ellipsis" :title="b.name">
                  <code>{{ b.name }}</code>
                </div>
              </td>
              <td>{{ fmtSize(b.size) }}</td>
              <td class="c-col-date">
                <div class="c-cell-ellipsis">{{ b.mod_time?.slice(0, 19) || '—' }}</div>
              </td>
              <td class="c-col-actions">
                <div class="c-cell-actions">
                  <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="restore(b)">恢复</button>
                  <a
                    class="c-btn c-btn--ghost c-btn--sm"
                    :href="`/api/v1/admin/backups/${encodeURIComponent(b.name)}`"
                    target="_blank"
                    rel="noopener"
                  >
                    下载
                  </a>
                  <select
                    v-if="configs.some((c) => c.enabled)"
                    class="c-input"
                    style="width: 120px; height: 30px"
                    @change="
                      (e) => {
                        const id = Number((e.target as HTMLSelectElement).value)
                        if (id) uploadToCloud(b.name, id)
                        ;(e.target as HTMLSelectElement).value = ''
                      }
                    "
                  >
                    <option value="">上传…</option>
                    <option v-for="c in configs.filter((x) => x.enabled)" :key="c.id" :value="c.id">
                      {{ c.name }}
                    </option>
                  </select>
                  <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="remove(b.name)">删除</button>
                </div>
              </td>
            </tr>
          </AdminTable>
        </section>

        <!-- 云端 -->
        <section v-show="tab === 'cloud'" class="panel-card">
          <div class="panel-toolbar">
            <span class="panel-meta">已配置 {{ configs.length }} 个</span>
            <div class="actions-row">
              <button type="button" class="c-btn c-btn--primary c-btn--sm" @click="openNew">添加配置</button>
              <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="loadCloud">刷新</button>
            </div>
          </div>

          <div v-if="cloudLoading" class="c-empty">加载中…</div>
          <div v-else-if="!configs.length" class="c-card c-card__body">
            <p class="c-empty" style="margin: 0; padding: 12px 0">暂无配置</p>
          </div>
          <div v-else class="config-list">
            <div
              v-for="c in configs"
              :key="c.id"
              class="config-card"
              :class="{ 'config-card--on': c.enabled }"
            >
              <div class="config-card__main">
                <div class="config-card__title">
                  <strong>{{ c.name }}</strong>
                  <span class="c-tag">{{ c.enabled ? '启用' : '禁用' }}</span>
                </div>
                <div class="config-card__meta">
                  <div>{{ c.webdav_url || '—' }}</div>
                  <div>{{ c.webdav_path || '/nav_backups/' }}</div>
                  <div>备份类型：{{ kindsText(c) }}</div>
                  <div v-if="c.auto_backup">
                    自动：每 {{ c.backup_interval || 24 }} 小时 · 保留 {{ c.backup_keep_count || 10 }} 份
                  </div>
                  <div>
                    {{ c.last_backup_time?.slice(0, 19) || '—' }} · {{ statusText(c.last_backup_status) }}
                  </div>
                </div>
              </div>
              <div class="config-card__actions">
                <button
                  type="button"
                  class="c-btn c-btn--primary c-btn--sm"
                  :disabled="!c.enabled || runningCloud === c.id"
                  @click="runCloudBackup(c.id)"
                >
                  {{ runningCloud === c.id ? '备份中…' : '立即备份' }}
                </button>
                <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="testConfig(c.id)">连接</button>
                <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="openEdit(c)">编辑</button>
                <button type="button" class="c-btn c-btn--danger c-btn--sm" @click="deleteConfig(c.id)">删除</button>
              </div>
            </div>
          </div>
        </section>
      </div>
    </div>

    <div v-if="showEditor" class="modal-mask" @click.self="showEditor = false">
      <div class="modal">
        <div class="modal__head">
          <h3>{{ editor.id ? '编辑配置' : '添加配置' }}</h3>
          <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="showEditor = false">关闭</button>
        </div>
        <div class="modal__body c-form">
          <label>名称 <input v-model="editor.name" class="c-input" /></label>
          <label>WebDAV URL <input v-model="editor.webdav_url" class="c-input" placeholder="https://" /></label>
          <label>用户名 <input v-model="editor.webdav_username" class="c-input" /></label>
          <label>
            密码
            <input
              v-model="editor.webdav_password"
              class="c-input"
              type="password"
              :placeholder="editor.id ? '留空保留原密码' : ''"
              autocomplete="new-password"
            />
          </label>
          <label>路径 <input v-model="editor.webdav_path" class="c-input" /></label>
          <div class="kind-block">
            <div class="kind-block__label">备份内容</div>
            <div class="kind-row">
              <label class="row-check">
                <input v-model="editor.backup_data" type="checkbox" /> 数据备份
              </label>
              <label class="row-check">
                <input v-model="editor.backup_config" type="checkbox" /> 配置备份
              </label>
            </div>
          </div>
          <div class="c-form c-form--2col nested">
            <label class="row-check"><input v-model="editor.enabled" type="checkbox" /> 启用</label>
            <label class="row-check"><input v-model="editor.auto_backup" type="checkbox" /> 自动备份</label>
            <label>
              间隔（小时）
              <input v-model.number="editor.backup_interval" class="c-input" type="number" min="1" />
            </label>
            <label>
              保留份数
              <input v-model.number="editor.backup_keep_count" class="c-input" type="number" min="1" />
            </label>
          </div>
          <div class="actions-row">
            <button type="button" class="c-btn c-btn--primary" @click="saveConfig">保存</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-layout {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.panel-card {
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.panel-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}
.panel-meta {
  font-size: 12px;
  color: var(--console-text-3);
}
.actions-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.config-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.config-card {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  justify-content: space-between;
  align-items: flex-start;
  padding: 14px 16px;
  border-radius: var(--console-radius);
  border: 1px solid var(--console-border);
  border-left: 4px solid #6b7a8c;
  background: var(--console-surface);
}
.config-card--on {
  border-left-color: var(--console-success);
}
.config-card__title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}
.config-card__meta {
  font-size: 12px;
  color: var(--console-text-3);
  line-height: 1.55;
}
.config-card__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.row-check {
  display: flex !important;
  flex-direction: row !important;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}
.kind-block {
  margin: 4px 0 8px;
  padding: 12px 14px;
  border-radius: var(--console-radius);
  border: 1px solid var(--console-border);
  background: var(--console-surface-2, rgba(0, 0, 0, 0.15));
}
.kind-block__label {
  font-size: 12px;
  font-weight: 600;
  color: var(--console-text-2);
  margin-bottom: 10px;
}
.kind-row {
  display: flex;
  flex-wrap: wrap;
  gap: 16px 24px;
}
.c-form--2col {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.c-form--2col.nested {
  margin: 0;
}
.modal-mask {
  position: fixed;
  inset: 0;
  z-index: 2000;
  background: rgba(0, 0, 0, 0.55);
  display: grid;
  place-items: center;
  padding: 24px;
}
.modal {
  width: min(480px, 100%);
  max-height: 90vh;
  overflow: auto;
  background: var(--console-surface);
  border: 1px solid var(--console-border);
  border-radius: 12px;
}
.modal__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  border-bottom: 1px solid var(--console-border);
}
.modal__head h3 {
  margin: 0;
  font-size: 15px;
}
.modal__body {
  padding: 16px;
}
</style>
