import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiGet, apiPost, setCsrfToken } from '@/shared/api/client'
import type { User } from '@/shared/types/models'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const loaded = ref(false)
  /** coalesce concurrent /auth/me (router + App + pages) */
  let fetchMeInflight: Promise<void> | null = null

  const isAdmin = computed(() => user.value?.role === 'admin' || user.value?.role === 'superadmin')
  const isSuper = computed(() => user.value?.role === 'superadmin')
  const isLoggedIn = computed(() => !!user.value)

  async function fetchMe() {
    if (fetchMeInflight) return fetchMeInflight
    fetchMeInflight = (async () => {
      try {
        const data = await apiGet<{ user: User | null; csrf_token: string }>('/api/v1/auth/me')
        user.value = data.user
        setCsrfToken(data.csrf_token || '')
      } catch {
        user.value = null
        setCsrfToken('')
      } finally {
        loaded.value = true
        fetchMeInflight = null
      }
    })()
    return fetchMeInflight
  }

  async function login(username: string, password: string, remember = true) {
    const data = await apiPost<{ user: User; csrf_token: string }>('/api/v1/auth/login', {
      username,
      password,
      remember,
    })
    user.value = data.user
    setCsrfToken(data.csrf_token || '')
    return data.user
  }

  async function register(payload: {
    username: string
    email: string
    password: string
    invitation_code: string
  }) {
    return apiPost('/api/v1/auth/register', payload)
  }

  async function logout() {
    try {
      await apiPost('/api/v1/auth/logout')
    } finally {
      user.value = null
      setCsrfToken('')
    }
  }

  return { user, loaded, isAdmin, isSuper, isLoggedIn, fetchMe, login, register, logout }
})
