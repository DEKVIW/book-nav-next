<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppIcon from '@/shared/ui/AppIcon.vue'
import AdminTable from '@/modules/admin/components/AdminTable.vue'
import { useToast } from '@/shared/composables/useToast'
import { deleteSearchShare, listSearchShares, revokeSearchShare } from '@/shared/api/searchShares'
import type { SearchShareRecord, SearchShareStatus } from '@/shared/types/searchShare'

const toast = useToast()
const items = ref<SearchShareRecord[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const query = ref('')
const status = ref<SearchShareStatus | ''>('')

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))

function isExpired(share: SearchShareRecord) {
  return share.status === 'active' && !!share.expires_at && new Date(share.expires_at).getTime() <= Date.now()
}

function statusLabel(share: SearchShareRecord) {
  if (isExpired(share)) return '已过期'
  if (share.status === 'revoked') return '已撤销'
  if (share.status === 'deleted') return '已删除'
  return '有效'
}

function statusClass(share: SearchShareRecord) {
  if (isExpired(share) || share.status === 'revoked' || share.status === 'deleted') return 'c-tag--danger'
  return 'c-tag--ok'
}

function formatDate(value?: string | null) {
  if (!value) return '永久'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '—' : date.toLocaleString('zh-CN', { hour12: false })
}

function shareUrl(share: SearchShareRecord) {
  return `${window.location.origin}/share/${encodeURIComponent(share.token)}`
}

async function load() {
  loading.value = true
  try {
    const data = await listSearchShares({ page: page.value, page_size: pageSize.value, status: status.value, q: query.value })
    items.value = data.items || []
    total.value = data.total || 0
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '分享列表加载失败')
  } finally {
    loading.value = false
  }
}

async function copy(share: SearchShareRecord) {
  try {
    await navigator.clipboard.writeText(shareUrl(share))
    toast.success('分享链接已复制')
  } catch {
    toast.error('复制失败')
  }
}

function open(share: SearchShareRecord) {
  window.open(shareUrl(share), '_blank', 'noopener,noreferrer')
}

async function revoke(share: SearchShareRecord) {
  if (share.status !== 'active' || isExpired(share)) return
  if (!window.confirm(`撤销「${share.title}」？撤销后链接将立即失效。`)) return
  try {
    await revokeSearchShare(share.id)
    toast.success('分享已撤销')
    await load()
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '撤销失败')
  }
}

async function remove(share: SearchShareRecord) {
  if (!window.confirm(`删除「${share.title}」？此操作不可恢复。`)) return
  try {
    await deleteSearchShare(share.id)
    toast.success('分享已删除')
    if (items.value.length === 1 && page.value > 1) page.value -= 1
    await load()
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '删除失败')
  }
}

function search() {
  page.value = 1
  void load()
}

onMounted(() => void load())
</script>

<template>
  <div class="admin-page shares-page">
    <header class="page-header">
      <div>
        <h1>分享管理</h1>
        <p class="page-header__hint">管理已生成的普通搜索和 AI 搜索结果分享链接。</p>
      </div>
    </header>

    <div class="c-toolbar shares-toolbar">
      <input v-model="query" class="c-input" placeholder="搜索标题或搜索词" @keyup.enter="search" />
      <select v-model="status" class="c-input" @change="search">
        <option value="">全部状态</option>
        <option value="active">有效</option>
        <option value="revoked">已撤销</option>
        <option value="deleted">已删除</option>
      </select>
      <button type="button" class="c-btn" @click="search"><AppIcon name="search" :size="15" /> 搜索</button>
    </div>

    <AdminTable :loading="loading" :is-empty="!items.length" empty="暂无分享记录">
      <template #head>
        <tr>
          <th>分享内容</th>
          <th class="shares-col-mode">模式</th>
          <th class="shares-col-count">结果</th>
          <th class="shares-col-time">有效期</th>
          <th class="shares-col-status">状态</th>
          <th class="c-col-actions">操作</th>
        </tr>
      </template>
      <tr v-for="share in items" :key="share.id">
        <td>
          <div class="c-cell-title" :title="share.title">{{ share.title }}</div>
          <span class="c-cell-sub" :title="share.query">
            {{ share.query }} · {{ share.creator_type === 'guest' ? '游客创建' : '管理员创建' }}
            <span v-if="share.include_private" class="shares-private-mark">含私人链接</span>
          </span>
        </td>
        <td class="shares-col-mode">
          <span class="c-tag" :class="share.mode === 'ai' ? 'c-tag--ai' : 'c-tag--ok'">{{ share.mode === 'ai' ? 'AI' : '普通' }}</span>
        </td>
        <td class="shares-col-count">{{ share.item_count }} 条 · {{ share.view_count }} 次访问</td>
        <td class="shares-col-time">{{ formatDate(share.expires_at) }}</td>
        <td class="shares-col-status"><span class="c-tag" :class="statusClass(share)">{{ statusLabel(share) }}</span></td>
        <td class="c-col-actions">
          <div class="c-cell-actions">
            <button type="button" class="c-btn c-btn--sm c-btn--ghost" title="复制链接" @click="copy(share)"><AppIcon name="link" :size="14" /> 复制</button>
            <button type="button" class="c-btn c-btn--sm c-btn--ghost" title="打开分享" @click="open(share)"><AppIcon name="external-link" :size="14" /> 打开</button>
            <button v-if="share.status === 'active' && !isExpired(share)" type="button" class="c-btn c-btn--sm c-btn--ghost" @click="revoke(share)">撤销</button>
            <button type="button" class="c-btn c-btn--sm c-btn--danger" @click="remove(share)"><AppIcon name="trash-2" :size="14" /> 删除</button>
          </div>
        </td>
      </tr>
      <template #footer>
        <span>第 {{ page }} / {{ totalPages }} 页 · {{ total }} 条</span>
        <div class="c-pagination__btns">
          <button type="button" class="c-btn c-btn--sm" :disabled="page <= 1" @click="page--; load()">上一页</button>
          <button type="button" class="c-btn c-btn--sm" :disabled="page >= totalPages" @click="page++; load()">下一页</button>
        </div>
      </template>
    </AdminTable>
  </div>
</template>

<style scoped>
.page-header__hint { margin: 6px 0 0; color: var(--text-muted); font-size: 13px; }
.shares-toolbar { flex-wrap: wrap; }
.shares-toolbar .c-input:first-child { min-width: min(320px, 100%); flex: 1; }
.shares-col-mode { width: 82px; }
.shares-col-count { width: 142px; color: var(--text-secondary); font-size: 12px; }
.shares-col-time { width: 170px; color: var(--text-secondary); font-size: 12px; }
.shares-col-status { width: 90px; }
.c-tag--ai { color: var(--magenta); border-color: color-mix(in srgb, var(--magenta) 45%, transparent); background: color-mix(in srgb, var(--magenta) 9%, transparent); }
.shares-private-mark { color: var(--console-warning); }
@media (max-width: 860px) {
  .shares-col-count, .shares-col-time { display: none; }
}
</style>
