<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { apiGet, apiPost, apiPut } from '@/shared/api/client'
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
}

const toast = useToast()
const tab = ref<'settings' | 'tasks'>('settings')
const jobs = ref<Job[]>([])
const running = ref(false)
const saving = ref(false)
const loading = ref(true)

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

async function loadSettings() {
  try {
    const raw = await apiGet<Record<string, unknown>>('/api/v1/admin/settings/icon')
    for (const [k, v] of Object.entries(raw || {})) {
      if (k in icon) (icon as any)[k] = v
    }
  } catch {
    /* defaults */
  }
}

async function loadJobs() {
  jobs.value = ((await apiGet('/api/v1/admin/jobs')) || []).filter((j: Job) => j.type === 'icon_sync')
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

async function start() {
  try {
    const j = await apiPost<Job>('/api/v1/admin/jobs/icons')
    toast.success(`任务 #${j.id} 已启动`)
    running.value = true
    tab.value = 'tasks'
    const poll = async () => {
      const cur = await apiGet<Job>(`/api/v1/admin/jobs/${j.id}`)
      await loadJobs()
      if (cur.status === 'completed' || cur.status === 'failed') {
        running.value = false
        toast.success(cur.status === 'completed' ? '图标任务完成' : '图标任务失败')
        return
      }
      setTimeout(poll, 1500)
    }
    poll()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '启动失败')
  }
}

onMounted(async () => {
  loading.value = true
  try {
    await Promise.all([loadSettings(), loadJobs()])
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
        <p>显示策略、自动抓取与批量任务</p>
      </div>
      <div class="page-header__actions">
        <button
          v-if="tab === 'settings'"
          type="button"
          class="c-btn c-btn--primary"
          :disabled="saving"
          @click="saveSettings"
        >
          {{ saving ? '保存中…' : '保存设置' }}
        </button>
        <button type="button" class="c-btn c-btn--primary" :disabled="running" @click="start">
          {{ running ? '运行中…' : '批量抓取' }}
        </button>
      </div>
    </header>

    <div v-if="loading" class="c-empty">加载中…</div>
    <template v-else>
      <div class="settings-layout">
        <div class="c-tabs" role="tablist">
          <button
            type="button"
            class="c-tabs__item"
            :class="{ active: tab === 'settings' }"
            @click="tab = 'settings'"
          >
            全局设置
          </button>
          <button
            type="button"
            class="c-tabs__item"
            :class="{ active: tab === 'tasks' }"
            @click="tab = 'tasks'"
          >
            批量任务
          </button>
        </div>

        <div class="settings-panel">
          <section v-show="tab === 'settings'" class="c-card c-card__body panel-card">
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
              <label class="row-check">
                <input v-model="icon.auto_fetch" type="checkbox" /> 新建链接时自动获取图标
              </label>
              <label class="row-check">
                <input v-model="icon.sync_local" type="checkbox" /> 抓取成功后保存到本地
              </label>
              <label class="row-check">
                <input v-model="icon.sync_imagebed" type="checkbox" /> 抓取成功后上传图床
              </label>
            </div>

            <div class="nested-card">
              <h4 class="nested-card__title">图床（可选）</h4>
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
                    :placeholder="icon.imagebed_token_configured ? '已配置，留空不修改' : ''"
                    autocomplete="new-password"
                  />
                </label>
              </div>
            </div>
          </section>

          <section v-show="tab === 'tasks'" class="panel-card">
            <AdminTable :is-empty="!jobs.length" empty="暂无图标任务">
              <template #head>
                <tr>
                  <th class="c-col-id">ID</th>
                  <th>状态</th>
                  <th>进度</th>
                  <th>成功</th>
                  <th>失败</th>
                </tr>
              </template>
              <tr v-for="j in jobs" :key="j.id">
                <td>{{ j.id }}</td>
                <td><span class="c-tag">{{ j.status }}</span></td>
                <td>{{ j.progress }}/{{ j.total }}</td>
                <td>{{ j.success }}</td>
                <td>{{ j.failed }}</td>
              </tr>
            </AdminTable>
          </section>
        </div>
      </div>
    </template>
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
}
.row-check {
  display: flex !important;
  flex-direction: row !important;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding-top: 22px;
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
@media (max-width: 960px) {
  .c-form--2col {
    grid-template-columns: 1fr;
  }
  .row-check {
    padding-top: 0;
  }
}
</style>
