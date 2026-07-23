<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useAuthStore } from '@/shared/stores/auth'
import { useToast } from '@/shared/composables/useToast'

const auth = useAuthStore()
const toast = useToast()
const router = useRouter()

const form = ref({
  username: '',
  email: '',
  password: '',
  invitation_code: '',
})
const loading = ref(false)

async function onSubmit() {
  loading.value = true
  try {
    await auth.register(form.value)
    toast.success('注册成功，请登录')
    router.push('/login')
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '注册失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <div class="auth-panel panel">
      <p class="eyebrow">ENLIST</p>
      <h1>邀请码注册</h1>
      <form class="auth-form" @submit.prevent="onSubmit">
        <label>
          <span>用户名</span>
          <input v-model="form.username" class="m-input" required />
        </label>
        <label>
          <span>邮箱</span>
          <input v-model="form.email" class="m-input" type="email" required />
        </label>
        <label>
          <span>密码</span>
          <input v-model="form.password" class="m-input" type="password" minlength="6" required />
        </label>
        <label>
          <span>邀请码</span>
          <input v-model="form.invitation_code" class="m-input" required />
        </label>
        <button class="m-btn m-btn--primary" type="submit" :disabled="loading">
          {{ loading ? '提交中…' : '注册' }}
        </button>
      </form>
      <p class="auth-footer">
        <RouterLink to="/login">返回登录</RouterLink>
        ·
        <RouterLink to="/">返回导航</RouterLink>
      </p>
    </div>
  </div>
</template>

<style scoped>
.auth-page {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: var(--space-6);
  background: var(--bg-void);
}
.auth-panel {
  width: min(400px, 100%);
  padding: var(--space-6);
}
.eyebrow {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--text-xs);
  letter-spacing: 0.16em;
  color: var(--glow-magenta);
}
h1 {
  margin: 8px 0 16px;
  font-family: var(--font-display);
}
.auth-form {
  display: grid;
  gap: 12px;
}
.auth-form label {
  display: grid;
  gap: 6px;
  font-size: var(--text-xs);
  color: var(--text-muted);
}
.auth-footer {
  margin-top: 16px;
  font-size: var(--text-xs);
  color: var(--text-muted);
}
.auth-footer a {
  color: var(--glow-cyan);
  text-decoration: none;
}
</style>
