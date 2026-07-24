<script setup lang="ts">
/**
 * 图标管理：显示策略 + 来源提供方 + 图床 + 批量抓取
 */
import { onMounted, reactive, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { apiGet, apiPost, apiPut } from '@/shared/api/client'
import { useToast } from '@/shared/composables/useToast'

interface Job {
  id: number
  type: string
  status: string
  progress: number
  total: number
  success: number
  failed: number
}

interface SourceProvider {
  id: string
  label: string
  kind: string
  builtin?: boolean
  enabled: boolean
  order: number
  supports_download?: boolean
  description?: string
  template?: string
}

const toast = useToast()
const running = ref(false)
const saving = ref(false)
const loading = ref(true)
const lastJob = ref<Job | null>(null)

const icon = reactive({
  display_mode: 'smart',
  auto_fetch: true,
  sync_local: true,
  sync_imagebed: false,
  imagebed_provider: '',
  imagebed_api_url: '',
  imagebed_token: '',
  imagebed_token_configured: false,
})

const providers = ref<SourceProvider[]>([])

async function loadSettings() {
  const raw = await apiGet<Record<string, unknown>>('/api/v1/admin/settings/icon')
  for (const [k, v] of Object.entries(raw || {})) {
    if (k === 'source_providers' && Array.isArray(v)) {
      providers.value = (v as SourceProvider[]).slice().sort((a, b) => (a.order ?? 0) - (b.order ?? 0))
      continue
    }
    if (k in icon) (icon as Record<string, unknown>)[k] = v
  }
}

async function loadLastJob() {
  try {
    const list = ((await apiGet('/api/v1/admin/jobs')) || []) as Job[]
    lastJob.value = list.find((j) => j.type === 'icon_sync') || null
    if (lastJob.value && (lastJob.value.status === 'running' || lastJob.value.status === 'pending')) {
      running.value = true
    } else {
      running.value = false
    }
  } catch {
    lastJob.value = null
  }
}

async function saveSettings() {
  saving.value = true
  try {
    const body: Record<string, unknown> = {
      display_mode: icon.display_mode,
      auto_fetch: icon.auto_fetch,
      sync_local: icon.sync_local,
      sync_imagebed: icon.sync_imagebed,
      imagebed_provider: icon.imagebed_provider,
      imagebed_api_url: icon.imagebed_api_url,
      source_providers: providers.value.map((p, i) => ({
        ...p,
        order: (i + 1) * 10,
      })),
    }
    const tok = String(icon.imagebed_token || '')
    if (tok && !tok.startsWith('****') && tok !== '********') {
      body.imagebed_token = tok
    }
    await apiPut('/api/v1/admin/settings/icon', body)
    toast.success('图标设置已保存')
    await loadSettings()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

function moveProvider(idx: number, dir: -1 | 1) {
  const j = idx + dir
  if (j < 0 || j >= providers.value.length) return
  const arr = providers.value.slice()
  const t = arr[idx]
  arr[idx] = arr[j]
  arr[j] = t
  providers.value = arr.map((p, i) => ({ ...p, order: (i + 1) * 10 }))
}

async function startBatch() {
  try {
    const j = await apiPost<Job>('/api/v1/admin/jobs/icons')
    toast.success(`任务 #${j.id} 已启动`)
    running.value = true
    lastJob.value = j
    const poll = async () => {
      try {
        const cur = await apiGet<Job>(`/api/v1/admin/jobs/${j.id}`)
        lastJob.value = cur
        if (cur.status === 'completed' || cur.status === 'failed' || cur.status === 'cancelled') {
          running.value = false
          const msg =
            cur.status === 'completed' ? '图标抓取完成' : cur.status === 'cancelled' ? '已停止' : '图标抓取失败'
          toast.success(msg)
          return
        }
        setTimeout(poll, 2000)
      } catch {
        running.value = false
      }
    }
    poll()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '启动失败')
  }
}

onMounted(async () => {
  loading.value = true
  try {
    await Promise.all([loadSettings(), loadLastJob()])
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="admin-page">
    <header class="page-header">
      <div>
        <h1>图标管理</h1>
        <p>显示策略、来源提供方与批量抓取</p>
      </div>
      <div class="page-header__actions">
        <RouterLink class="c-btn c-btn--ghost c-btn--sm" to="/admin/jobs">任务中心</RouterLink>
        <button type="button" class="c-btn c-btn--ghost" :disabled="saving" @click="saveSettings">
          {{ saving ? '保存中…' : '保存设置' }}
        </button>
        <button type="button" class="c-btn c-btn--primary" :disabled="running" @click="startBatch">
          {{ running ? '抓取中…' : '批量抓取' }}
        </button>
      </div>
    </header>

    <div v-if="loading" class="c-empty">加载中…</div>
    <template v-else>
      <section class="c-card c-card__body">
        <h3 class="c-card__title">显示与自动抓取</h3>
        <div class="c-form c-form--2col">
          <label>
            前台显示优先级
            <select v-model="icon.display_mode" class="c-input">
              <option value="smart">智能（优先本地，自动回退）</option>
              <option value="source">优先源站图标</option>
              <option value="local">优先本地缓存</option>
              <option value="imagebed">优先图床</option>
            </select>
          </label>
          <div class="check-stack">
            <label class="row-check">
              <input v-model="icon.auto_fetch" type="checkbox" />
              新建链接时自动获取图标
            </label>
            <label class="row-check">
              <input v-model="icon.sync_local" type="checkbox" />
              抓取成功后保存到本地
            </label>
            <label class="row-check">
              <input v-model="icon.sync_imagebed" type="checkbox" />
              抓取成功后上传图床
            </label>
          </div>
        </div>
      </section>

      <section class="c-card c-card__body">
        <h3 class="c-card__title">来源提供方</h3>
        <p class="field-hint">按顺序尝试：原站解析 → 已启用的代理服务。仅启用且靠前的会参与抓取。</p>
        <div class="provider-list">
          <div v-for="(p, idx) in providers" :key="p.id" class="provider-row">
            <label class="row-check provider-enable">
              <input v-model="p.enabled" type="checkbox" />
            </label>
            <div class="provider-meta">
              <div class="provider-name">
                <strong>{{ p.label }}</strong>
                <span class="provider-id">{{ p.id }}</span>
                <span class="c-tag">{{ p.kind === 'origin' ? '原站' : '代理' }}</span>
              </div>
              <div class="provider-desc">{{ p.description || p.template || '—' }}</div>
            </div>
            <div class="provider-actions">
              <button type="button" class="c-btn c-btn--ghost c-btn--sm" :disabled="idx === 0" @click="moveProvider(idx, -1)">
                上移
              </button>
              <button
                type="button"
                class="c-btn c-btn--ghost c-btn--sm"
                :disabled="idx === providers.length - 1"
                @click="moveProvider(idx, 1)"
              >
                下移
              </button>
            </div>
          </div>
        </div>
      </section>

      <section class="c-card c-card__body">
        <h3 class="c-card__title">图床（可选）</h3>
        <p class="field-hint">开启「上传图床」后生效；Token 已配置时留空表示不修改。</p>
        <div class="c-form c-form--2col">
          <label>
            提供方
            <input v-model="icon.imagebed_provider" class="c-input" placeholder="如 lsky / smms" />
          </label>
          <label>
            API 地址
            <input v-model="icon.imagebed_api_url" class="c-input" placeholder="https://" />
          </label>
          <label class="span-2">
            Token
            <input
              v-model="icon.imagebed_token"
              class="c-input"
              type="password"
              :placeholder="icon.imagebed_token_configured ? '已配置，留空不修改' : '可选'"
              autocomplete="new-password"
            />
          </label>
        </div>
      </section>

      <section v-if="lastJob" class="c-card c-card__body">
        <div class="job-card__row">
          <div>
            <h3 class="c-card__title" style="margin-bottom: 4px">最近图标任务</h3>
            <p class="field-hint" style="margin: 0">
              #{{ lastJob.id }} · {{ lastJob.status }} · {{ lastJob.progress }}/{{ lastJob.total }} · 成功
              {{ lastJob.success }} / 失败 {{ lastJob.failed }}
            </p>
          </div>
          <RouterLink class="c-btn c-btn--ghost c-btn--sm" to="/admin/jobs">查看全部</RouterLink>
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
.c-form--2col {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px 18px;
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
.check-stack {
  display: flex;
  flex-direction: column;
  gap: 10px;
  justify-content: center;
}
.row-check {
  display: flex !important;
  flex-direction: row !important;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  font-size: 13px;
  color: var(--console-text);
}
.field-hint {
  margin: 0 0 12px;
  font-size: 12px;
  color: var(--console-text-3);
  line-height: 1.45;
}
.provider-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.provider-row {
  display: grid;
  grid-template-columns: 28px 1fr auto;
  gap: 12px;
  align-items: center;
  padding: 10px 12px;
  border: 1px solid var(--console-border);
  border-radius: var(--console-radius, 8px);
  background: rgba(0, 0, 0, 0.12);
}
.provider-enable {
  padding: 0 !important;
}
.provider-name {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 2px;
}
.provider-id {
  font-family: var(--font-mono, ui-monospace, monospace);
  font-size: 11px;
  color: var(--console-text-3);
}
.provider-desc {
  font-size: 12px;
  color: var(--console-text-3);
  word-break: break-all;
}
.provider-actions {
  display: flex;
  gap: 6px;
}
.job-card__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
@media (max-width: 960px) {
  .c-form--2col {
    grid-template-columns: 1fr;
  }
  .provider-row {
    grid-template-columns: 28px 1fr;
  }
  .provider-actions {
    grid-column: 2;
  }
}
</style>
