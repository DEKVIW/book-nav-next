<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import { apiGet, apiPost } from '@/shared/api/client'
import { useToast } from '@/shared/composables/useToast'

const toast = useToast()
const legacyFile = ref('')
const legacyMode = ref<'merge' | 'replace'>('replace')
const importing = ref(false)

async function doExport() {
  const data = await apiGet('/api/v1/admin/export')
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = `booknav-export-${Date.now()}.json`
  a.click()
}

async function doLegacyImport() {
  if (!legacyFile.value.trim()) {
    toast.error('请填写文件名')
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
  await apiPost('/api/v1/admin/data/clear-websites')
  toast.success('已清空')
}
</script>

<template>
  <div class="admin-page">
    <header class="page-header">
      <div>
        <h1>数据管理</h1>
        <p>导入导出与运维入口</p>
      </div>
    </header>

    <div class="entry-grid">
      <RouterLink class="entry-card" to="/admin/deadlinks">
        <strong>死链检测</strong>
        <span>检查失效链接</span>
      </RouterLink>
      <RouterLink class="entry-card" to="/admin/icons">
        <strong>图标管理</strong>
        <span>图标策略与任务</span>
      </RouterLink>
      <RouterLink class="entry-card" to="/admin/backups">
        <strong>备份管理</strong>
        <span>本地与云端备份</span>
      </RouterLink>
      <RouterLink class="entry-card" to="/admin/logs">
        <strong>操作日志</strong>
        <span>变更记录</span>
      </RouterLink>
    </div>

    <section class="c-card c-card__body">
      <h3 class="c-card__title">导入</h3>
      <p class="field-hint">将 .db3 文件放入 data 目录后填写文件名</p>
      <div class="row">
        <input v-model="legacyFile" class="c-input" placeholder="export.db3" style="max-width: 280px" />
        <select v-model="legacyMode" class="c-input" style="max-width: 120px">
          <option value="replace">替换</option>
          <option value="merge">合并</option>
        </select>
        <button type="button" class="c-btn c-btn--primary" :disabled="importing" @click="doLegacyImport">
          {{ importing ? '导入中…' : '导入' }}
        </button>
      </div>
    </section>

    <section class="c-card c-card__body">
      <h3 class="c-card__title">导出</h3>
      <button type="button" class="c-btn c-btn--ghost" @click="doExport">导出 JSON</button>
    </section>

    <section class="c-card c-card__body">
      <h3 class="c-card__title">危险操作</h3>
      <button type="button" class="c-btn c-btn--danger" @click="clearSites">清空全部链接</button>
    </section>
  </div>
</template>

<style scoped>
.entry-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
}
.entry-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 14px 16px;
  border-radius: var(--console-radius);
  border: 1px solid var(--console-border);
  background: var(--console-surface);
  text-decoration: none;
  color: inherit;
}
.entry-card:hover {
  border-color: rgba(110, 142, 251, 0.4);
}
.entry-card strong {
  font-size: 14px;
}
.entry-card span {
  font-size: 12px;
  color: var(--console-text-3);
}
.row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}
.field-hint {
  margin: 0 0 12px;
  font-size: 12px;
  color: var(--console-text-3);
}
</style>
