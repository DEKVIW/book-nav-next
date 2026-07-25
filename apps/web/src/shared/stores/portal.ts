import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiGet, apiPost, apiPatch, apiDelete, apiPut } from '@/shared/api/client'
import type { Category, HomeData, PublicSettings, Website } from '@/shared/types/models'

export type SearchMeta = {
  mode?: string
  ai?: boolean
  stage?: string
  summary?: string
  refined?: boolean
}

export const usePortalStore = defineStore('portal', () => {
  const categories = ref<Category[]>([])
  const featured = ref<Website[]>([])
  const settings = ref<PublicSettings | null>(null)
  const loading = ref(false)
  const error = ref('')
  const searchResults = ref<Website[] | null>(null)
  const searchQuery = ref('')
  const searchMeta = ref<SearchMeta | null>(null)
  const searchLoading = ref(false)
  let loadHomeInflight: Promise<void> | null = null
  let searchSeq = 0
  let searchES: EventSource | null = null

  async function loadHome() {
    if (loadHomeInflight) return loadHomeInflight
    if (!categories.value.length) loading.value = true
    error.value = ''
    loadHomeInflight = (async () => {
      try {
        const data = await apiGet<HomeData>('/api/v1/portal/home')
        categories.value = data.categories || []
        featured.value = data.featured || []
        settings.value = data.settings
      } catch (e: unknown) {
        error.value = e instanceof Error ? e.message : '加载失败'
      } finally {
        loading.value = false
        loadHomeInflight = null
      }
    })()
    return loadHomeInflight
  }

  function stopSearchStream() {
    if (searchES) {
      searchES.close()
      searchES = null
    }
  }

  function applySearchPayload(
    seq: number,
    data: {
      websites?: Website[]
      mode?: string
      ai?: boolean
      stage?: string
      summary?: string
      refined?: boolean
    },
  ) {
    if (seq !== searchSeq) return
    searchResults.value = data.websites || []
    searchMeta.value = {
      mode: data.mode,
      ai: data.ai,
      stage: data.stage,
      summary: data.summary,
      refined: data.refined,
    }
    if (data.stage === 'final' || data.stage === 'error') {
      searchLoading.value = false
      stopSearchStream()
    }
  }

  /**
   * Non-AI: one-shot JSON.
   * AI: two-stage SSE (initial fusion → optional final rerank).
   * searchSeq drops stale in-flight responses.
   */
  async function search(q: string, ai = false) {
    const query = q.trim()
    searchQuery.value = query
    stopSearchStream()
    const seq = ++searchSeq
    if (!query) {
      searchResults.value = null
      searchMeta.value = null
      searchLoading.value = false
      return
    }
    searchLoading.value = true
    searchMeta.value = { stage: 'loading' }

    if (!ai) {
      try {
        const data = await apiGet<{
          websites?: Website[]
          mode?: string
          ai?: boolean
          summary?: string
          refined?: boolean
        }>(`/api/v1/portal/search?q=${encodeURIComponent(query)}&ai=0`)
        if (seq !== searchSeq) return
        applySearchPayload(seq, { ...data, stage: 'final' })
      } catch (e) {
        if (seq !== searchSeq) return
        searchResults.value = []
        searchMeta.value = { stage: 'error' }
        searchLoading.value = false
        throw e
      }
      return
    }

    await new Promise<void>((resolve, reject) => {
      const url = `/api/v1/portal/search/stream?q=${encodeURIComponent(query)}&ai=1`
      const es = new EventSource(url)
      searchES = es
      let settled = false
      const done = (err?: Error) => {
        if (settled) return
        settled = true
        stopSearchStream()
        if (seq === searchSeq) searchLoading.value = false
        if (err) reject(err)
        else resolve()
      }
      es.onmessage = (ev) => {
        if (seq !== searchSeq) {
          es.close()
          return
        }
        try {
          const data = JSON.parse(ev.data) as {
            websites?: Website[]
            mode?: string
            ai?: boolean
            stage?: string
            summary?: string
            refined?: boolean
            error?: string
          }
          if (data.stage === 'error') {
            if (!searchResults.value) searchResults.value = []
            searchMeta.value = { stage: 'error' }
            done(new Error(data.error || '搜索失败'))
            return
          }
          applySearchPayload(seq, data)
          if (data.stage === 'final') done()
        } catch (e) {
          done(e instanceof Error ? e : new Error('parse error'))
        }
      }
      es.onerror = () => {
        if (searchResults.value && seq === searchSeq) {
          searchMeta.value = { ...(searchMeta.value || {}), stage: 'final' }
          done()
          return
        }
        done(new Error('搜索连接中断'))
      }
    })
  }

  function clearSearch() {
    searchSeq++
    stopSearchStream()
    searchQuery.value = ''
    searchResults.value = null
    searchMeta.value = null
    searchLoading.value = false
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

  async function translateText(text: string, field: 'title' | 'description' = 'description') {
    return apiPost<{ text: string; original: string; field: string }>(
      '/api/v1/portal/utils/translate',
      { text, field },
    )
  }

  async function enhanceSiteInfo(payload: { url?: string; title?: string; description?: string }) {
    return apiPost<{ title?: string; description?: string }>(
      '/api/v1/portal/utils/site-info',
      payload,
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
    searchMeta,
    searchLoading,
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
    translateText,
    enhanceSiteInfo,
    visit,
    loadCategoryAll,
  }
})
