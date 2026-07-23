<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { apiGet, apiPatch } from '@/shared/api/client'
import { useToast } from '@/shared/composables/useToast'
import type { User } from '@/shared/types/models'
import AdminTable from '../components/AdminTable.vue'

const toast = useToast()
const users = ref<User[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    users.value = await apiGet('/api/v1/admin/users')
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

async function setRole(u: User, role: string) {
  try {
    await apiPatch(`/api/v1/admin/users/${u.id}`, { role })
    toast.success('角色已更新')
    await load()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '失败')
  }
}

onMounted(load)
</script>

<template>
  <div>
    <header class="page-header">
      <div>
        <h1>用户管理</h1>
        <p>角色：user / admin / superadmin</p>
      </div>
    </header>

    <AdminTable :loading="loading" :is-empty="!users.length" empty="暂无用户">
      <template #head>
        <tr>
          <th class="c-col-id">ID</th>
          <th>用户名</th>
          <th>邮箱</th>
          <th style="width: 160px">角色</th>
          <th class="c-col-actions">操作</th>
        </tr>
      </template>
      <tr v-for="u in users" :key="u.id">
        <td>{{ u.id }}</td>
        <td class="c-cell-title">{{ u.username }}</td>
        <td>
          <div class="c-cell-ellipsis" :title="u.email">{{ u.email }}</div>
        </td>
        <td>
          <span class="c-tag">{{ u.role }}</span>
        </td>
        <td class="c-col-actions">
          <div class="c-cell-actions">
            <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="setRole(u, 'user')">user</button>
            <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="setRole(u, 'admin')">admin</button>
            <button type="button" class="c-btn c-btn--ghost c-btn--sm" @click="setRole(u, 'superadmin')">super</button>
          </div>
        </td>
      </tr>
    </AdminTable>
  </div>
</template>
