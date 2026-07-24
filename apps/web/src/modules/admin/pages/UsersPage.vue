<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { apiDelete, apiGet, apiPatch, apiPostForm } from '@/shared/api/client'
import { useToast } from '@/shared/composables/useToast'
import { useAuthStore } from '@/shared/stores/auth'
import type { Role, User } from '@/shared/types/models'
import AdminTable from '../components/AdminTable.vue'

const toast = useToast()
const auth = useAuthStore()
const users = ref<User[]>([])
const loading = ref(false)
const saving = ref(false)

const showForm = ref(false)
const editing = ref<User | null>(null)
const form = ref({
  username: '',
  email: '',
  role: 'user' as Role,
  new_password: '',
})
const avatarFile = ref<File | null>(null)
const avatarPreview = ref<string>('')

const roleLabels: Record<Role, string> = {
  user: '普通用户',
  admin: '管理员',
  superadmin: '超级管理员',
}

const meId = computed(() => auth.user?.id)

function letter(name: string) {
  return (name || '?').trim().slice(0, 1).toUpperCase()
}

function avatarSrc(u: User | null | undefined) {
  if (!u?.avatar) return ''
  return u.avatar
}

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

function openEdit(u: User) {
  editing.value = u
  form.value = {
    username: u.username,
    email: u.email,
    role: u.role,
    new_password: '',
  }
  avatarFile.value = null
  avatarPreview.value = avatarSrc(u)
  showForm.value = true
}

function closeForm() {
  if (avatarPreview.value.startsWith('blob:')) {
    URL.revokeObjectURL(avatarPreview.value)
  }
  showForm.value = false
  editing.value = null
  avatarFile.value = null
  avatarPreview.value = ''
}

function onAvatarPick(ev: Event) {
  const input = ev.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (!file.type.startsWith('image/')) {
    toast.error('请选择图片文件')
    input.value = ''
    return
  }
  if (file.size > 5 * 1024 * 1024) {
    toast.error('头像不能超过 5MB')
    input.value = ''
    return
  }
  if (avatarPreview.value.startsWith('blob:')) {
    URL.revokeObjectURL(avatarPreview.value)
  }
  avatarFile.value = file
  avatarPreview.value = URL.createObjectURL(file)
}

async function save() {
  if (!editing.value) return
  const id = editing.value.id
  const username = form.value.username.trim()
  const email = form.value.email.trim()
  if (!username) {
    toast.error('用户名不能为空')
    return
  }
  if (!email) {
    toast.error('邮箱不能为空')
    return
  }
  if (form.value.new_password && form.value.new_password.length < 6) {
    toast.error('密码至少 6 位')
    return
  }
  saving.value = true
  try {
    const body: Record<string, unknown> = {
      username,
      email,
      role: form.value.role,
    }
    if (form.value.new_password) {
      body.new_password = form.value.new_password
    }
    await apiPatch(`/api/v1/admin/users/${id}`, body)

    if (avatarFile.value) {
      const fd = new FormData()
      fd.append('avatar', avatarFile.value)
      await apiPostForm(`/api/v1/admin/users/${id}/avatar`, fd)
    }

    toast.success('用户信息已更新')
    closeForm()
    await load()
    // refresh session user if we edited ourselves
    if (meId.value === id) {
      try {
        await auth.fetchMe()
      } catch {
        /* ignore */
      }
    }
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function removeUser(u: User) {
  if (u.id === meId.value) {
    toast.error('不能删除当前登录的用户')
    return
  }
  if (!confirm(`确定删除用户「${u.username}」？此操作不可恢复。`)) return
  try {
    await apiDelete(`/api/v1/admin/users/${u.id}`)
    toast.success('用户已删除')
    if (editing.value?.id === u.id) closeForm()
    await load()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '删除失败')
  }
}

function formatTime(s?: string) {
  if (!s) return '—'
  try {
    const d = new Date(s)
    if (Number.isNaN(d.getTime())) return s
    return d.toLocaleString()
  } catch {
    return s
  }
}

onMounted(load)
</script>

<template>
  <div class="admin-page">
    <header class="page-header">
      <h1>用户管理</h1>
    </header>

    <div v-if="showForm && editing" class="c-card c-card__body user-edit">
      <div class="user-edit__head">
        <h3 class="c-card__title">编辑用户</h3>
        <span class="c-tag">ID {{ editing.id }}</span>
      </div>

      <div class="user-edit__grid">
        <div class="user-edit__avatar-col">
          <div class="user-avatar-lg" :class="{ 'is-admin': editing.role !== 'user' }">
            <img
              v-if="avatarPreview"
              :src="avatarPreview"
              alt=""
              @error="($event.target as HTMLImageElement).style.display = 'none'"
            />
            <span v-else>{{ letter(form.username || editing.username) }}</span>
          </div>
          <label class="user-avatar-pick">
            <input type="file" accept="image/*" hidden @change="onAvatarPick" />
            <span class="c-btn c-btn--sm c-btn--ghost">上传头像</span>
          </label>
          <p class="user-edit__hint">JPG / PNG / GIF / WebP，最大 5MB</p>
        </div>

        <div class="c-form-2 user-edit__fields">
          <label>
            用户名
            <input v-model="form.username" class="c-input" autocomplete="username" />
          </label>
          <label>
            邮箱
            <input v-model="form.email" class="c-input" type="email" autocomplete="email" />
          </label>
          <label>
            新密码
            <input
              v-model="form.new_password"
              class="c-input"
              type="password"
              autocomplete="new-password"
              placeholder="留空则不修改"
            />
          </label>
          <label>
            角色
            <select v-model="form.role" class="c-input">
              <option value="user">普通用户</option>
              <option value="admin">管理员</option>
              <option value="superadmin">超级管理员</option>
            </select>
          </label>
          <div class="span-2 user-edit__meta">
            <span>注册：{{ formatTime(editing.created_at) }}</span>
            <span v-if="editing.id === meId" class="c-tag c-tag--ok">当前登录</span>
          </div>
        </div>
      </div>

      <div class="user-edit__actions">
        <button type="button" class="c-btn c-btn--primary" :disabled="saving" @click="save">
          {{ saving ? '保存中…' : '保存' }}
        </button>
        <button type="button" class="c-btn c-btn--ghost" :disabled="saving" @click="closeForm">取消</button>
        <button
          v-if="editing.id !== meId"
          type="button"
          class="c-btn c-btn--danger c-btn--sm"
          :disabled="saving"
          style="margin-left: auto"
          @click="removeUser(editing)"
        >
          删除用户
        </button>
      </div>
    </div>

    <AdminTable :loading="loading" :is-empty="!users.length" empty="暂无用户">
      <template #head>
        <tr>
          <th class="c-col-icon" />
          <th class="c-col-id">ID</th>
          <th>用户名</th>
          <th>邮箱</th>
          <th style="width: 140px">角色</th>
          <th class="c-col-actions">操作</th>
        </tr>
      </template>
      <tr v-for="u in users" :key="u.id">
        <td class="c-col-icon">
          <img
            v-if="u.avatar"
            class="c-avatar-sm"
            :src="u.avatar"
            alt=""
            loading="lazy"
          />
          <span
            v-else
            class="c-avatar-fallback"
            :class="{ 'c-avatar-fallback--admin': u.role !== 'user' }"
          >{{ letter(u.username) }}</span>
        </td>
        <td>{{ u.id }}</td>
        <td class="c-cell-title">
          {{ u.username }}
          <span v-if="u.id === meId" class="c-tag c-tag--ok" style="margin-left: 6px">我</span>
        </td>
        <td>
          <div class="c-cell-ellipsis" :title="u.email">{{ u.email }}</div>
        </td>
        <td>
          <span
            class="c-tag"
            :class="{
              'c-tag--warn': u.role === 'admin',
              'c-tag--danger': u.role === 'superadmin',
            }"
          >{{ roleLabels[u.role] || u.role }}</span>
        </td>
        <td class="c-col-actions">
          <div class="c-cell-actions">
            <button type="button" class="c-btn c-btn--sm c-btn--ghost" @click="openEdit(u)">编辑</button>
            <button
              v-if="u.id !== meId"
              type="button"
              class="c-btn c-btn--sm c-btn--ghost"
              @click="removeUser(u)"
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
.user-edit {
  margin: 0;
}
.user-edit__head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
}
.user-edit__grid {
  display: grid;
  grid-template-columns: 160px 1fr;
  gap: 20px;
  align-items: start;
}
@media (max-width: 720px) {
  .user-edit__grid {
    grid-template-columns: 1fr;
  }
}
.user-edit__avatar-col {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  text-align: center;
}
.user-avatar-lg {
  width: 112px;
  height: 112px;
  border-radius: 50%;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 40px;
  font-weight: 700;
  color: #fff;
  background: linear-gradient(135deg, #3498db, #5ebbff);
  border: 3px solid rgba(255, 255, 255, 0.12);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.25);
}
.user-avatar-lg.is-admin {
  background: linear-gradient(135deg, #6a11cb, #845ec2);
}
.user-avatar-lg img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.user-avatar-pick {
  cursor: pointer;
}
.user-edit__hint {
  margin: 0;
  font-size: 11px;
  color: var(--console-text-3);
  line-height: 1.4;
}
.user-edit__fields {
  max-width: none;
}
.user-edit__meta {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 12px;
  color: var(--console-text-3);
  padding-top: 4px;
}
.user-edit__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 16px;
  align-items: center;
}
.c-avatar-fallback--admin {
  background: linear-gradient(135deg, #6a11cb, #845ec2) !important;
  color: #fff !important;
}
</style>
