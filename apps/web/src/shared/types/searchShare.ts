import type { Website } from './models'

export type SearchShareMode = 'keyword' | 'ai'
export type SearchShareStatus = 'active' | 'revoked' | 'deleted'

export interface SearchShareRecord {
  id: number
  token: string
  created_by?: number
  creator_name?: string
  creator_type: 'admin' | 'guest'
  title: string
  query: string
  mode: SearchShareMode
  summary?: string
  status: SearchShareStatus
  include_private: boolean
  expires_at?: string | null
  view_count: number
  item_count: number
  created_at: string
  updated_at: string
}

export interface CreateSearchShareResponse {
  id: number
  token: string
  share_path: string
  item_count: number
  excluded_count: number
  expires_at?: string | null
}

export interface PublicSearchShareResponse {
  share: {
    title: string
    query: string
    mode: SearchShareMode
    summary?: string
    status: SearchShareStatus
    expires_at?: string | null
    view_count: number
    item_count: number
    created_at: string
  }
  websites: Website[]
}
