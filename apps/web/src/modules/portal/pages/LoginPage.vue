<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/shared/stores/auth'
import { useToast } from '@/shared/composables/useToast'

const auth = useAuthStore()
const toast = useToast()
const router = useRouter()
const route = useRoute()

const username = ref('')
const password = ref('')
const remember = ref(true)
const loading = ref(false)

async function onSubmit() {
  loading.value = true
  try {
    await auth.login(username.value, password.value, remember.value)
    toast.success('登录成功')
    const redirect = (route.query.redirect as string) || '/'
    router.push(redirect)
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <div class="auth-panel hull hull--elevated">
      <div class="hull-corners" aria-hidden="true">
        <span class="c-tl" /><span class="c-tr" /><span class="c-bl" /><span class="c-br" />
      </div>
      <h1>登录</h1>
      <form class="auth-form" @submit.prevent="onSubmit">
        <label>
          <span>用户名</span>
          <input v-model="username" class="m-input" type="text" autocomplete="username" required />
        </label>
        <label>
          <span>密码</span>
          <input
            v-model="password"
            class="m-input"
            type="password"
            autocomplete="current-password"
            required
          />
        </label>
        <label class="remember">
          <input v-model="remember" type="checkbox" /> 记住我
        </label>
        <button class="m-btn m-btn--primary" type="submit" :disabled="loading">
          {{ loading ? '登录中…' : '登录' }}
        </button>
      </form>
      <p class="auth-footer">
        <RouterLink to="/">返回首页</RouterLink>
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
}
.auth-panel {
  width: min(400px, 100%);
  padding: var(--space-6);
}
.auth-panel h1 {
  margin: 0 0 1.25rem;
  font-size: 1.35rem;
}
.auth-form {
  display: grid;
  gap: 12px;
}
.auth-form label {
  display: grid;
  gap: 6px;
  font-size: 13px;
}
.remember {
  display: flex !important;
  flex-direction: row !important;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}
.auth-footer {
  margin: 1rem 0 0;
  font-size: 13px;
  opacity: 0.75;
  text-align: center;
}
</style>
