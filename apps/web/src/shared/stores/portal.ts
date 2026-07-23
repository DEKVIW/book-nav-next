import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiGet, apiPost, apiPatch, apiDelete, apiPut } from '@/shared/api/client'
import type { Category, HomeData, PublicSettings, Website } from '@/shared/types/models'

export const usePortalStore = defineStore('portal', () => {
  const categories = ref<Category[]>([])
  const featured = ref<Website[]>([])
  const settings = ref<PublicSettings | null>(null)
  const loading = ref(false)
  const error = ref('')
  const searchResults = ref<Website[] | null>(null)
  const searchQuery = ref('')

  async function loadHome() {
    loading.value = true
    error.value = ''
    try {
      const data = await apiGet<HomeData>('/api/v1/portal/home')
      categories.value = data.categories || []
      featured.value = data.featured || []
      settings.value = data.settings
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : '加载失败'
    } finally {
      loading.value = false
    }
  }

  async function search(q: string, ai = false) {
    searchQuery.value = q
    if (!q.trim()) {
      searchResults.value = null
      return
    }
    const data = await apiGet<{ websites: Website[] }>(
      `/api/v1/portal/search?q=${encodeURIComponent(q)}&ai=${ai ? 1 : 0}`,
    )
    searchResults.value = data.websites || []
  }

  function clearSearch() {
    searchQuery.value = ''
    searchResults.value = null
  }

  async function createWebsite(payload: Record<string, unknown>) {
    const site = await apiPost<Website>('/api/v1/portal/websites', payload)
    await loadHome()
    return site
  }

  async function updateWebsite(id: number, payload: Record<string, unknown>) {
    const site = await apiPatch<Website>(`/api/v1/portal/websites/${id}`, payload)
    await loadHome()
    return site
  }

  async function deleteWebsite(id: number) {
    await apiDelete(`/api/v1/portal/websites/${id}`)
    await loadHome()
  }

  async function reorderWebsites(categoryId: number | null, ids: number[]) {
    await apiPut('/api/v1/portal/websites/order', {
      category_id: categoryId,
      ids,
    })
  }

  async function reorderCategories(ids: number[]) {
    await apiPut('/api/v1/portal/categories/order', { ids })
    await loadHome()
  }

  async function checkUrl(url: string) {
    return apiGet<{ exists: boolean; website: Website | null }>(
      `/api/v1/portal/utils/check-url?url=${encodeURIComponent(url)}`,
    )
  }

  async function fetchSite(url: string) {
    return apiGet<Record<string, unknown>>(
      `/api/v1/portal/utils/fetch-site?url=${encodeURIComponent(url)}`,
    )
  }

  async function visit(id: number) {
    return apiPost<{
      website: Website
      enable_transition: boolean
      countdown: number
    }>(`/api/v1/portal/websites/${id}/visit`)
  }

  async function loadCategoryAll(id: number) {
    return apiGet<Website[]>(`/api/v1/portal/categories/${id}/websites`)
  }

  return {
    categories,
    featured,
    settings,
    loading,
    error,
    searchResults,
    searchQuery,
    loadHome,
    search,
    clearSearch,
    createWebsite,
    updateWebsite,
    deleteWebsite,
    reorderWebsites,
    reorderCategories,
    checkUrl,
    fetchSite,
    visit,
    loadCategoryAll,
  }
})
