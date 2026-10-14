import { apiDelete, apiGet, apiPost } from './client'
import type {
  CreateSearchShareResponse,
  PublicSearchShareResponse,
  SearchShareRecord,
  SearchShareStatus,
} from '../types/searchShare'

export function createSearchShare(payload: {
  title: string
  query: string
  mode: 'keyword' | 'ai'
  summary: string
  website_ids: number[]
  expires_in_days?: number
  include_private?: boolean
}) {
  return apiPost<CreateSearchShareResponse>('/api/v1/shares', payload)
}

export function listSearchShares(params: { page: number; page_size: number; status?: SearchShareStatus | ''; q?: string }) {
  const query = new URLSearchParams({
    page: String(params.page),
    page_size: String(params.page_size),
  })
  if (params.status) query.set('status', params.status)
  if (params.q?.trim()) query.set('q', params.q.trim())
  return apiGet<{ items: SearchShareRecord[]; total: number; page: number; page_size: number }>(
    `/api/v1/admin/shares?${query.toString()}`,
  )
}

export function revokeSearchShare(id: number) {
  return apiPost<void>(`/api/v1/admin/shares/${id}/revoke`)
}

export function deleteSearchShare(id: number) {
  return apiDelete<void>(`/api/v1/admin/shares/${id}`)
}

export function getPublicSearchShare(token: string) {
  return apiGet<PublicSearchShareResponse>(`/api/v1/shares/${encodeURIComponent(token)}`)
}
