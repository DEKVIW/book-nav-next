<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { apiGet } from '@/shared/api/client'
import { useAuthStore } from '@/shared/stores/auth'
import { useToast } from '@/shared/composables/useToast'

const auth = useAuthStore()
const toast = useToast()
const stats = ref<Record<string, number>>({})
const loading = ref(true)

onMounted(async () => {
  try {
    stats.value = await apiGet('/api/v1/admin/stats')
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
})

const labels: Record<string, string> = {
  users: '用户',
  websites: '链接',
  categories: '分类',
  jobs_running: '运行中任务',
}

const quick = [
  { to: '/admin/websites', title: '链接管理', desc: '管理站点链接' },
  { to: '/admin/categories', title: '分类管理', desc: '分类结构与排序' },
  { to: '/admin/settings', title: '站点设置', desc: '基本信息与检索', super: true },
  { to: '/admin/data', title: '数据管理', desc: '导入导出与运维', super: true },
  { to: '/admin/icons', title: '图标管理', desc: '图标策略与批量任务', super: true },
  { to: '/admin/backups', title: '备份管理', desc: '本地与云端备份', super: true },
  { to: '/admin/jobs', title: '任务', desc: '后台任务进度' },
]
</script>

<template>
  <div class="admin-page">
    <header class="page-header">
      <div>
        <h1>仪表盘</h1>
        <p>数据概览与常用入口</p>
      </div>
    </header>

    <div v-if="loading" class="c-empty">加载中…</div>
    <template v-else>
      <div class="c-stats">
        <div v-for="(v, k) in stats" :key="k" class="c-stat">
          <span class="c-stat__label">{{ labels[k] || k }}</span>
          <div class="c-stat__value">{{ v }}</div>
        </div>
      </div>

      <h3 class="c-section-title">快捷入口</h3>
      <div class="c-quick">
        <RouterLink
          v-for="q in quick.filter((i) => !i.super || auth.isSuper)"
          :key="q.to"
          :to="q.to"
        >
          <strong>{{ q.title }}</strong>
          <span>{{ q.desc }}</span>
        </RouterLink>
      </div>
    </template>
  </div>
</template>
