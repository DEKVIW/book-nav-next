<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { apiDelete, apiGet, apiPost, apiPut } from '@/shared/api/client'
import { useToast } from '@/shared/composables/useToast'

const toast = useToast()
const route = useRoute()
const router = useRouter()
const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const indexing = ref(false)
const detecting = ref(false)
const testingTasks = ref(false)
const testResult = ref<{ ok?: boolean; message?: string; details?: Record<string, string> } | null>(null)

type TabKey = 'basic' | 'transition' | 'ai' | 'vector'
const tabs: { key: TabKey; label: string }[] = [
  { key: 'basic', label: '基本设置' },
  { key: 'transition', label: '过渡页' },
  { key: 'ai', label: 'AI 配置' },
  { key: 'vector', label: '向量配置' },
]
const tabKeys = new Set(tabs.map((t) => t.key))
const tab = ref<TabKey>('basic')

function applyTabFromQuery() {
  const q = String(route.query.tab || '')
  if (tabKeys.has(q as TabKey)) tab.value = q as TabKey
}

function setTab(key: TabKey) {
  tab.value = key
  router.replace({ query: { ...route.query, tab: key } })
}

watch(() => route.query.tab, applyTabFromQuery)

const site = reactive({
  name: '',
  subtitle: '',
  footer: '',
  logo: '',
  favicon: '',
  keywords: '',
  description: '',
})
const transition = reactive({
  enable: false,
  time: 5,
  admin_time: 0,
  remember_choice: true,
  show_description: true,
  theme: 'default',
  color: '#6e8efb',
  ad1: '',
  ad2: '',
})
const announcement = reactive({
  enabled: false,
  title: '',
  content: '',
  start: '',
  end: '',
  remember_days: 7,
})
const aiFlags = reactive({
  enabled: false,
  allow_anonymous: false,
  temperature: 0.7,
  max_tokens: 800,
})
const vector = reactive({
  enabled: false,
  qdrant_url: 'http://localhost:6333',
  collection: 'websites',
  embedding_api_base_url: '',
  embedding_api_key: '',
  embedding_model: 'text-embedding-3-small',
  dimension: 1536,
  similarity_threshold: 0.3,
  max_results: 50,
  embedding_api_key_configured: false,
})

interface AIProvider {
  id: number
  name: string
  api_base_url: string
  api_key?: string
  api_key_configured?: boolean
  interface_mode: string
  enabled: boolean
  priority: number
  model_catalog?: { id: string; compatible?: string }[]
  recommended_models?: Record<string, string>
  probe_last_at?: string
  probe_error?: string
}
interface TaskBinding {
  mode: string
  provider_id: number | null
  model_name: string
}
interface TaskTest {
  status: string
  message: string
  provider_name?: string
  model_name?: string
  tested_at?: string
}

const providers = ref<AIProvider[]>([])
const taskBindings = reactive<Record<string, TaskBinding>>({
  intent: { mode: 'auto', provider_id: null, model_name: '' },
  rerank: { mode: 'auto', provider_id: null, model_name: '' },
  translate: { mode: 'auto', provider_id: null, model_name: '' },
  site_info: { mode: 'auto', provider_id: null, model_name: '' },
})
const taskTests = ref<Record<string, TaskTest>>({})
const effective = ref<Record<string, Record<string, unknown>>>({})
const summary = ref({ provider_count: 0, enabled_provider_count: 0, detected_provider_count: 0 })

const taskLabels: Record<string, string> = {
  intent: '搜索意图分析',
  rerank: '搜索结果重排',
  translate: '翻译',
  site_info: '网站信息补全',
}

// modals
const showProviders = ref(false)
const showBindings = ref(false)
const showOverview = ref(false)
const showAdvanced = ref(false)
const showEditor = ref(false)
const editor = reactive({
  id: 0,
  name: '',
  api_base_url: '',
  api_key: '',
  interface_mode: 'auto',
  enabled: true,
  priority: 100,
})

function applyNs(raw: Record<string, unknown>, target: Record<string, unknown>) {
  for (const [k, v] of Object.entries(raw || {})) {
    if (k in target) (target as any)[k] = v
  }
}

async function loadNs(ns: string) {
  return apiGet<Record<string, unknown>>(`/api/v1/admin/settings/${ns}`)
}

async function loadAIState() {
  const state = await apiGet<{
    providers: AIProvider[]
    task_bindings: Record<string, TaskBinding>
    task_test_results: Record<string, TaskTest>
    effective_tasks: Record<string, Record<string, unknown>>
    summary: typeof summary.value
    enabled: boolean
    allow_anonymous: boolean
    temperature: number
    max_tokens: number
  }>('/api/v1/admin/ai/state')
  providers.value = state.providers || []
  if (state.task_bindings) {
    for (const k of Object.keys(taskBindings)) {
      if (state.task_bindings[k]) Object.assign(taskBindings[k], state.task_bindings[k])
    }
  }
  taskTests.value = state.task_test_results || {}
  effective.value = state.effective_tasks || {}
  if (state.summary) summary.value = state.summary
  aiFlags.enabled = !!state.enabled
  aiFlags.allow_anonymous = !!state.allow_anonymous
  if (state.temperature != null) aiFlags.temperature = state.temperature
  if (state.max_tokens != null) aiFlags.max_tokens = state.max_tokens
}

async function load() {
  loading.value = true
  try {
    applyNs(await loadNs('site'), site)
    applyNs(await loadNs('transition'), transition)
    applyNs(await loadNs('announcement'), announcement)
    applyNs(await loadNs('vector'), vector)
    await loadAIState()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

function stripSecrets(obj: Record<string, unknown>, secretKeys: string[]) {
  const out: Record<string, unknown> = { ...obj }
  for (const k of secretKeys) {
    const v = String(out[k] ?? '')
    if (!v || v === '********' || v.startsWith('****')) delete out[k]
    delete out[`${k}_configured`]
  }
  for (const k of Object.keys(out)) {
    if (k.endsWith('_configured')) delete out[k]
  }
  return out
}

async function saveAll() {
  saving.value = true
  try {
    await apiPut('/api/v1/admin/settings/site', { ...site })
    await apiPut('/api/v1/admin/settings/transition', { ...transition })
    await apiPut('/api/v1/admin/settings/announcement', { ...announcement })
    await apiPut('/api/v1/admin/settings/ai', {
      enabled: aiFlags.enabled,
      allow_anonymous: aiFlags.allow_anonymous,
      temperature: aiFlags.temperature,
      max_tokens: aiFlags.max_tokens,
    })
    await apiPut('/api/v1/admin/settings/vector', stripSecrets({ ...vector }, ['embedding_api_key']))
    toast.success('设置已保存')
    await load()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

function openNewProvider() {
  Object.assign(editor, {
    id: 0,
    name: '',
    api_base_url: '',
    api_key: '',
    interface_mode: 'auto',
    enabled: true,
    priority: 100,
  })
  showEditor.value = true
}
function openEditProvider(p: AIProvider) {
  Object.assign(editor, {
    id: p.id,
    name: p.name,
    api_base_url: p.api_base_url,
    api_key: '',
    interface_mode: p.interface_mode || 'auto',
    enabled: p.enabled,
    priority: p.priority || 100,
  })
  showEditor.value = true
}

async function saveProvider() {
  try {
    const body: Record<string, unknown> = {
      id: editor.id || undefined,
      name: editor.name,
      api_base_url: editor.api_base_url,
      interface_mode: editor.interface_mode,
      enabled: editor.enabled,
      priority: editor.priority,
    }
    if (editor.api_key && !editor.api_key.startsWith('****')) body.api_key = editor.api_key
    await apiPost('/api/v1/admin/ai/providers', body)
    toast.success('提供方已保存')
    showEditor.value = false
    await loadAIState()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '保存失败')
  }
}

async function deleteProvider(id: number) {
  if (!confirm('确定删除该提供方？')) return
  try {
    await apiDelete(`/api/v1/admin/ai/providers/${id}`)
    toast.success('已删除')
    await loadAIState()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '删除失败')
  }
}

async function detectOne(id: number) {
  detecting.value = true
  try {
    await apiPost(`/api/v1/admin/ai/providers/${id}/detect`)
    toast.success('模型检测完成')
    await loadAIState()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '检测失败')
  } finally {
    detecting.value = false
  }
}

async function detectAll() {
  detecting.value = true
  try {
    const res = await apiPost<{ message: string; ok: boolean }>('/api/v1/admin/ai/providers/detect-all')
    if (res.ok) toast.success(res.message)
    else toast.error(res.message || '部分失败')
    await loadAIState()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '检测失败')
  } finally {
    detecting.value = false
  }
}

async function testOne(id: number) {
  try {
    const res = await apiPost<{ message: string }>(`/api/v1/admin/ai/providers/${id}/test`, {})
    toast.success(res.message || '测试成功')
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '测试失败')
  }
}

async function saveBindings() {
  try {
    await apiPut('/api/v1/admin/ai/task-bindings', { task_bindings: { ...taskBindings } })
    toast.success('任务模型已保存')
    await loadAIState()
    showBindings.value = false
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '保存失败')
  }
}

async function runTaskTests() {
  testingTasks.value = true
  try {
    const res = await apiPost<{ message: string; ok: boolean }>('/api/v1/admin/ai/test-tasks', {
      task_bindings: { ...taskBindings },
    })
    if (res.ok) toast.success(res.message)
    else toast.error(res.message)
    await loadAIState()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '测试失败')
  } finally {
    testingTasks.value = false
  }
}

async function testVector() {
  testing.value = true
  testResult.value = null
  try {
    const data = await apiPost<{
      ok: boolean
      message: string
      details: Record<string, string>
    }>('/api/v1/admin/vector/test', {
      qdrant_url: vector.qdrant_url,
      embedding_api_base_url: vector.embedding_api_base_url,
      embedding_api_key:
        vector.embedding_api_key && !String(vector.embedding_api_key).startsWith('****')
          ? vector.embedding_api_key
          : '',
      embedding_model: vector.embedding_model,
      collection: vector.collection,
    })
    testResult.value = data
    if (data.ok) toast.success(data.message || '测试通过')
    else toast.error(data.message || '测试失败')
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '测试失败')
  } finally {
    testing.value = false
  }
}

async function startVectorIndex() {
  indexing.value = true
  try {
    const j = await apiPost<{ id: number }>('/api/v1/admin/jobs/vector-index')
    toast.success(`向量索引任务 #${j.id} 已启动（任务中心查看进度）`)
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '启动失败')
  } finally {
    indexing.value = false
  }
}

const vectorReadyHint = computed(() => {
  if (!vector.enabled) return { level: 'info', text: '向量搜索未启用' }
  const hasEmbed =
    vector.embedding_api_base_url ||
    vector.embedding_api_key_configured ||
    (vector.embedding_api_key && !String(vector.embedding_api_key).startsWith('****')) ||
    summary.value.enabled_provider_count > 0
  if (vector.qdrant_url && vector.embedding_model && hasEmbed) {
    return { level: 'ok', text: `已配置 · ${vector.embedding_model}` }
  }
  return { level: 'warn', text: '配置不完整' }
})

function modelsForProvider(pid: number | null): string[] {
  if (!pid) return []
  const p = providers.value.find((x) => x.id === pid)
  if (!p?.model_catalog?.length) return []
  return p.model_catalog.map((m) => m.id)
}

function statusClass(st: string) {
  if (st === 'success') return 'chip chip--ok'
  if (st === 'error') return 'chip chip--err'
  return 'chip'
}

onMounted(() => {
  applyTabFromQuery()
  load()
})
</script>

<template>
  <div class="admin-page">
    <header class="page-header">
      <h1>站点设置</h1>
      <div class="page-header__actions">
        <button type="button" class="c-btn c-btn--primary" :disabled="saving || loading" @click="saveAll">
          {{ saving ? '保存中…' : '保存设置' }}
        </button>
      </div>
    </header>

    <div v-if="loading" class="c-empty">加载中…</div>
    <template v-else>
      <div class="settings-layout">
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

        <div class="settings-panel">
          <!-- 基本设置 -->
          <section v-show="tab === 'basic'" class="c-card c-card__body panel-card">
            <h3 class="c-card__title">基本信息</h3>
            <div class="c-form c-form--2col">
              <label>
                站点名称
                <input v-model="site.name" class="c-input" />
              </label>
              <label>
                副标题
                <input v-model="site.subtitle" class="c-input" />
              </label>
              <label>
                SEO 关键词
                <input v-model="site.keywords" class="c-input" placeholder="逗号分隔" />
              </label>
              <label>
                SEO 描述
                <input v-model="site.description" class="c-input" />
              </label>
              <label>
                Logo URL
                <input v-model="site.logo" class="c-input" placeholder="https://" />
              </label>
              <label>
                Favicon URL
                <input v-model="site.favicon" class="c-input" placeholder="https://" />
              </label>
              <label class="span-2">
                页脚（HTML）
                <textarea v-model="site.footer" class="c-input" rows="4" />
              </label>
            </div>

            <div class="nested-card">
              <h4 class="nested-card__title">弹窗公告</h4>
              <div class="c-form c-form--2col">
                <label class="row-check span-2">
                  <input v-model="announcement.enabled" type="checkbox" /> 启用公告
                </label>
                <label class="span-2">
                  标题
                  <input v-model="announcement.title" class="c-input" />
                </label>
                <label class="span-2">
                  内容
                  <textarea
                    v-model="announcement.content"
                    class="c-input"
                    rows="6"
                    placeholder="支持 Markdown 或 HTML。样式由前台机甲主题统一覆盖。"
                  />
                </label>
                <label>
                  开始时间
                  <input v-model="announcement.start" class="c-input" type="datetime-local" />
                </label>
                <label>
                  结束时间
                  <input v-model="announcement.end" class="c-input" type="datetime-local" />
                </label>
                <label>
                  不再提示（天）
                  <input
                    v-model.number="announcement.remember_days"
                    class="c-input"
                    type="number"
                    min="1"
                    max="365"
                  />
                </label>
              </div>
            </div>
          </section>

          <!-- 过渡页 -->
          <section v-show="tab === 'transition'" class="c-card c-card__body panel-card">
            <h3 class="c-card__title">过渡页</h3>
            <div class="c-form c-form--2col">
              <label class="row-check span-2">
                <input v-model="transition.enable" type="checkbox" /> 启用过渡页
              </label>
              <label>
                访客停留（秒，0=直跳）
                <input v-model.number="transition.time" class="c-input" type="number" min="0" max="30" />
              </label>
              <label>
                管理员停留（秒）
                <input v-model.number="transition.admin_time" class="c-input" type="number" min="0" max="30" />
              </label>
              <label class="row-check">
                <input v-model="transition.remember_choice" type="checkbox" /> 允许不再显示
              </label>
              <label class="row-check">
                <input v-model="transition.show_description" type="checkbox" /> 显示网站描述
              </label>
              <label>
                主题
                <select v-model="transition.theme" class="c-input">
                  <option value="default">默认</option>
                  <option value="minimal">极简</option>
                  <option value="dark">深色</option>
                  <option value="card">卡片</option>
                </select>
              </label>
              <label>
                主色
                <input v-model="transition.color" class="c-input" type="color" style="height: 40px; padding: 4px" />
              </label>
              <label class="span-2">
                广告位 1
                <textarea v-model="transition.ad1" class="c-input" rows="3" />
              </label>
              <label class="span-2">
                广告位 2
                <textarea v-model="transition.ad2" class="c-input" rows="3" />
              </label>
            </div>
          </section>

          <!-- AI 配置 -->
          <section v-show="tab === 'ai'" class="c-card c-card__body panel-card">
            <h3 class="c-card__title">AI 配置</h3>
            <div class="switch-row">
              <label class="row-check">
                <input v-model="aiFlags.enabled" type="checkbox" /> 启用 AI 搜索
              </label>
              <label class="row-check">
                <input v-model="aiFlags.allow_anonymous" type="checkbox" /> 允许未登录使用
              </label>
            </div>

            <div class="actions-row">
              <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="showProviders = true">
                提供方
              </button>
              <button type="button" class="c-btn c-btn--primary c-btn--sm" :disabled="detecting" @click="detectAll">
                {{ detecting ? '检测中…' : '自动选型' }}
              </button>
              <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="showBindings = true">任务模型</button>
              <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="showOverview = true">总览</button>
              <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="showAdvanced = true">高级</button>
              <button
                type="button"
                class="c-btn c-btn--ghost c-btn--sm"
                :disabled="testingTasks"
                @click="runTaskTests"
              >
                {{ testingTasks ? '测试中…' : '任务测试' }}
              </button>
            </div>

            <div class="chips">
              <span class="chip">提供方 {{ summary.provider_count }}</span>
              <span class="chip">已启用 {{ summary.enabled_provider_count }}</span>
              <span class="chip">已探测 {{ summary.detected_provider_count }}</span>
            </div>

            <div class="task-grid">
              <div v-for="(label, key) in taskLabels" :key="key" class="task-card">
                <div class="task-card__title">{{ label }}</div>
                <div class="task-card__meta">
                  <template v-if="effective[key]?.provider_name">
                    {{ effective[key].provider_name }} · {{ effective[key].model_name || '未选模型' }}
                  </template>
                  <template v-else>未绑定</template>
                </div>
                <span :class="statusClass(taskTests[key]?.status || 'idle')">
                  {{ taskTests[key]?.status || 'idle' }}
                </span>
                <div v-if="taskTests[key]?.message" class="task-card__msg">{{ taskTests[key].message }}</div>
              </div>
            </div>
          </section>

          <!-- 向量配置 -->
          <section v-show="tab === 'vector'" class="c-card c-card__body panel-card">
            <h3 class="c-card__title">向量配置</h3>
            <div class="c-form c-form--2col">
              <label class="row-check span-2">
                <input v-model="vector.enabled" type="checkbox" /> 启用向量搜索
              </label>
              <label class="span-2">
                Embedding API URL
                <input
                  v-model="vector.embedding_api_base_url"
                  class="c-input"
                  placeholder="留空则使用 AI 提供方"
                />
              </label>
              <label class="span-2">
                Embedding API Key
                <input
                  v-model="vector.embedding_api_key"
                  class="c-input"
                  type="password"
                  :placeholder="vector.embedding_api_key_configured ? '已配置，留空不修改' : '可选'"
                  autocomplete="new-password"
                />
              </label>
              <label>
                Embedding 模型
                <input v-model="vector.embedding_model" class="c-input" placeholder="text-embedding-3-small" />
              </label>
              <label>
                Qdrant 地址
                <input v-model="vector.qdrant_url" class="c-input" placeholder="http://localhost:6333" />
              </label>
              <label>
                Collection
                <input v-model="vector.collection" class="c-input" />
              </label>
              <label>
                向量维度
                <input v-model.number="vector.dimension" class="c-input" type="number" min="64" max="4096" />
              </label>
              <label>
                相似度阈值
                <input
                  v-model.number="vector.similarity_threshold"
                  class="c-input"
                  type="number"
                  min="0"
                  max="1"
                  step="0.05"
                />
              </label>
              <label>
                最大结果数
                <input v-model.number="vector.max_results" class="c-input" type="number" min="10" max="200" />
              </label>
            </div>
            <div
              class="status-banner"
              :class="{
                'status-banner--ok': vectorReadyHint.level === 'ok',
                'status-banner--warn': vectorReadyHint.level === 'warn',
                'status-banner--info': vectorReadyHint.level === 'info',
              }"
            >
              {{ vectorReadyHint.text }}
            </div>
            <div class="actions-row">
              <button type="button" class="c-btn c-btn--ghost" :disabled="testing" @click="testVector">
                {{ testing ? '测试中…' : '测试连接' }}
              </button>
              <button type="button" class="c-btn c-btn--primary" :disabled="indexing" @click="startVectorIndex">
                {{ indexing ? '启动中…' : '批量索引' }}
              </button>
            </div>
            <div v-if="testResult" class="test-box">
              <div class="test-box__title">{{ testResult.message }}</div>
              <ul v-if="testResult.details">
                <li v-for="(v, k) in testResult.details" :key="k"><strong>{{ k }}</strong>：{{ v }}</li>
              </ul>
            </div>
          </section>
        </div>
      </div>
    </template>

    <!-- Modal: providers list -->
    <div v-if="showProviders" class="modal-mask" @click.self="showProviders = false">
      <div class="modal modal--xl">
        <div class="modal__head">
          <h3>AI 提供方配置</h3>
          <div class="actions-row">
            <button type="button" class="c-btn c-btn--primary c-btn--sm" @click="openNewProvider">添加提供方</button>
            <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="showProviders = false">关闭</button>
          </div>
        </div>
        <div class="modal__body">
          <div class="table-wrap">
            <table class="mini-table">
              <thead>
                <tr>
                  <th>名称</th>
                  <th>URL</th>
                  <th>模式</th>
                  <th>优先级</th>
                  <th>状态</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="p in providers" :key="p.id">
                  <td>{{ p.name }} {{ p.enabled ? '' : '(停用)' }}</td>
                  <td class="ellip" :title="p.api_base_url">{{ p.api_base_url }}</td>
                  <td>{{ p.interface_mode }}</td>
                  <td>{{ p.priority }}</td>
                  <td>
                    <span v-if="p.probe_error" class="chip chip--err">失败</span>
                    <span v-else-if="p.model_catalog?.length" class="chip chip--ok">
                      {{ p.model_catalog.length }} 模型
                    </span>
                    <span v-else class="chip">未探测</span>
                  </td>
                  <td class="ops">
                    <button type="button" class="linkish" @click="openEditProvider(p)">编辑</button>
                    <button type="button" class="linkish" :disabled="detecting" @click="detectOne(p.id)">探测</button>
                    <button type="button" class="linkish" @click="testOne(p.id)">测试</button>
                    <button type="button" class="linkish danger" @click="deleteProvider(p.id)">删</button>
                  </td>
                </tr>
                <tr v-if="!providers.length">
                  <td colspan="6" class="empty-cell">还没有提供方，请先添加</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>

    <!-- Modal: provider editor -->
    <div v-if="showEditor" class="modal-mask" @click.self="showEditor = false">
      <div class="modal">
        <div class="modal__head">
          <h3>{{ editor.id ? '编辑提供方' : '添加提供方' }}</h3>
          <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="showEditor = false">关闭</button>
        </div>
        <div class="modal__body c-form">
          <label>名称 <input v-model="editor.name" class="c-input" placeholder="主线路 / 备用" /></label>
          <label>基础 URL <input v-model="editor.api_base_url" class="c-input" placeholder="https://api.openai.com" /></label>
          <label>
            API Key
            <input
              v-model="editor.api_key"
              class="c-input"
              type="password"
              :placeholder="editor.id ? '留空保留原密钥' : 'sk-...'"
              autocomplete="new-password"
            />
          </label>
          <div class="c-form c-form--2col nested">
            <label>
              接口模式
              <select v-model="editor.interface_mode" class="c-input">
                <option value="auto">自动兜底</option>
                <option value="chat">Chat</option>
                <option value="responses">Responses</option>
              </select>
            </label>
            <label>
              优先级
              <input v-model.number="editor.priority" class="c-input" type="number" min="1" />
            </label>
          </div>
          <label class="row-check"><input v-model="editor.enabled" type="checkbox" /> 启用</label>
          <div class="actions-row">
            <button type="button" class="c-btn c-btn--primary" @click="saveProvider">保存提供方</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Modal: task bindings -->
    <div v-if="showBindings" class="modal-mask" @click.self="showBindings = false">
      <div class="modal modal--xl">
        <div class="modal__head">
          <h3>四项任务模型配置</h3>
          <div class="actions-row">
            <button type="button" class="c-btn c-btn--primary c-btn--sm" @click="saveBindings">保存</button>
            <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="showBindings = false">关闭</button>
          </div>
        </div>
        <div class="modal__body">
          <div v-for="(label, key) in taskLabels" :key="key" class="bind-row">
            <strong>{{ label }}</strong>
            <select v-model="taskBindings[key].mode" class="c-input">
              <option value="auto">自动推荐</option>
              <option value="manual">手动指定</option>
            </select>
            <select
              v-model.number="taskBindings[key].provider_id"
              class="c-input"
              :disabled="taskBindings[key].mode !== 'manual'"
            >
              <option :value="null">选择提供方</option>
              <option v-for="p in providers" :key="p.id" :value="p.id">{{ p.name }}</option>
            </select>
            <select
              v-model="taskBindings[key].model_name"
              class="c-input"
              :disabled="taskBindings[key].mode !== 'manual'"
            >
              <option value="">选择模型</option>
              <option v-for="m in modelsForProvider(taskBindings[key].provider_id)" :key="m" :value="m">
                {{ m }}
              </option>
            </select>
          </div>
        </div>
      </div>
    </div>

    <!-- Modal: overview -->
    <div v-if="showOverview" class="modal-mask" @click.self="showOverview = false">
      <div class="modal modal--xl">
        <div class="modal__head">
          <h3>AI 配置总览</h3>
          <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="showOverview = false">关闭</button>
        </div>
        <div class="modal__body">
          <pre class="overview-pre">{{
            JSON.stringify(
              {
                enabled: aiFlags.enabled,
                allow_anonymous: aiFlags.allow_anonymous,
                summary,
                effective,
                providers: providers.map((p) => ({
                  id: p.id,
                  name: p.name,
                  url: p.api_base_url,
                  enabled: p.enabled,
                  priority: p.priority,
                  models: p.model_catalog?.length || 0,
                  recommended: p.recommended_models,
                })),
                bindings: taskBindings,
              },
              null,
              2,
            )
          }}</pre>
        </div>
      </div>
    </div>

    <!-- Modal: advanced -->
    <div v-if="showAdvanced" class="modal-mask" @click.self="showAdvanced = false">
      <div class="modal">
        <div class="modal__head">
          <h3>AI 高级设置</h3>
          <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="showAdvanced = false">完成</button>
        </div>
        <div class="modal__body c-form c-form--2col">
          <label>
            温度参数
            <input v-model.number="aiFlags.temperature" class="c-input" type="number" min="0" max="1" step="0.1" />
          </label>
          <label>
            最大 Token
            <input v-model.number="aiFlags.max_tokens" class="c-input" type="number" min="100" max="4000" />
          </label>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.row-check {
  display: flex !important;
  flex-direction: row !important;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}
.hint {
  margin: 0 0 8px;
  font-size: 12px;
  color: var(--console-text-3);
  line-height: 1.55;
}
.hint code {
  font-family: var(--console-mono);
  font-size: 11px;
  padding: 1px 5px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.06);
}
/* 固定 Tab + 面板间距，切换时不再跳动 */
.settings-layout {
  display: flex;
  flex-direction: column;
  gap: 16px;
  width: 100%;
  min-width: 0;
}
.settings-panel {
  width: 100%;
  min-width: 0;
}
.panel-card {
  margin: 0;
}
.panel-card .c-card__title {
  margin-bottom: 16px;
}
.c-form--2col {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px 16px;
}
.c-form--2col label {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 12px;
  color: var(--console-text-2);
}
.c-form--2col .span-2 {
  grid-column: 1 / -1;
}
.c-form--2col.nested {
  margin: 0;
  padding: 0;
}
.switch-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  margin-bottom: 14px;
}
.actions-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 12px 0;
}
.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.chip {
  display: inline-flex;
  padding: 3px 10px;
  border-radius: 999px;
  font-size: 11px;
  background: rgba(255, 255, 255, 0.06);
  color: var(--console-text-2);
}
.chip--ok {
  background: rgba(34, 197, 94, 0.15);
  color: #86efac;
}
.chip--err {
  background: rgba(239, 68, 68, 0.15);
  color: #fca5a5;
}
.task-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  margin-top: 8px;
}
.task-card {
  padding: 10px;
  border-radius: 8px;
  border: 1px solid var(--console-border);
  background: #0c1016;
}
.task-card__title {
  font-size: 12px;
  font-weight: 600;
  margin-bottom: 4px;
}
.task-card__meta {
  font-size: 11px;
  color: var(--console-text-3);
  margin-bottom: 6px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.task-card__msg {
  margin-top: 4px;
  font-size: 11px;
  color: var(--console-text-2);
  line-height: 1.4;
}
.status-banner {
  padding: 10px 12px;
  border-radius: var(--console-radius-sm);
  font-size: 12px;
  border: 1px solid var(--console-border);
  background: rgba(255, 255, 255, 0.03);
}
.status-banner--ok {
  border-color: rgba(34, 197, 94, 0.35);
  background: rgba(34, 197, 94, 0.08);
  color: #86efac;
}
.status-banner--warn {
  border-color: rgba(245, 158, 11, 0.35);
  background: rgba(245, 158, 11, 0.08);
  color: #fcd34d;
}
.status-banner--info {
  border-color: rgba(110, 142, 251, 0.3);
  background: rgba(110, 142, 251, 0.08);
  color: #a5b4fc;
}
.test-box {
  padding: 10px 12px;
  border-radius: var(--console-radius-sm);
  border: 1px solid var(--console-border);
  background: #0c1016;
  font-size: 12px;
}
.test-box__title {
  font-weight: 600;
  margin-bottom: 6px;
}
.test-box ul {
  margin: 0;
  padding-left: 18px;
  color: var(--console-text-2);
}

/* modals */
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
  width: min(520px, 100%);
  max-height: min(90vh, 800px);
  overflow: auto;
  background: var(--console-surface);
  border: 1px solid var(--console-border);
  border-radius: 12px;
  box-shadow: 0 24px 64px rgba(0, 0, 0, 0.5);
}
.modal--xl {
  width: min(960px, 100%);
}
.modal__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
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
.table-wrap {
  overflow: auto;
}
.mini-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
.mini-table th,
.mini-table td {
  padding: 8px 10px;
  border-bottom: 1px solid var(--console-border);
  text-align: left;
  vertical-align: middle;
}
.mini-table th {
  color: var(--console-text-3);
  font-weight: 600;
  font-size: 11px;
  text-transform: uppercase;
}
.ellip {
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ops {
  white-space: nowrap;
}
.linkish {
  background: none;
  border: none;
  color: #93c5fd;
  cursor: pointer;
  font-size: 12px;
  padding: 0 4px;
}
.linkish.danger {
  color: #fca5a5;
}
.empty-cell {
  text-align: center;
  color: var(--console-text-3);
  padding: 24px !important;
}
.bind-row {
  display: grid;
  grid-template-columns: 120px 1fr 1fr 1fr;
  gap: 10px;
  align-items: center;
  margin-bottom: 12px;
}
.bind-row strong {
  font-size: 12px;
}
.overview-pre {
  margin: 0;
  padding: 12px;
  border-radius: 8px;
  background: #0c1016;
  font-size: 11px;
  line-height: 1.45;
  overflow: auto;
  max-height: 60vh;
  color: var(--console-text-2);
  font-family: var(--console-mono);
}
@media (max-width: 960px) {
  .c-form--2col,
  .switch-row,
  .task-grid,
  .bind-row {
    grid-template-columns: 1fr;
  }
}
</style>
