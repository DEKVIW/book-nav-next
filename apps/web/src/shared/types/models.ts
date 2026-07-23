export type Role = 'user' | 'admin' | 'superadmin'

export interface User {
  id: number
  username: string
  email: string
  avatar?: string
  role: Role
}

export interface Website {
  id: number
  title: string
  url: string
  description?: string
  icon?: string
  category_id?: number | null
  created_by: number
  is_featured: boolean
  is_private: boolean
  sort_order: number
  views: number
  is_valid: boolean
  category_name?: string
  viewer_ids?: number[]
}

export interface Category {
  id: number
  name: string
  description?: string
  icon?: string
  color?: string
  sort_order: number
  display_limit: number
  parent_id?: number | null
  children?: Category[]
  websites?: Website[]
  website_count?: number
  direct_count?: number
  total_count_with_children?: number
  displayed_subcategory_id?: number | null
}

export interface PublicSettings {
  site_name: string
  site_subtitle?: string
  site_logo?: string
  site_favicon?: string
  footer_content?: string
  ai_search_enabled: boolean
  ai_search_allow_anonymous: boolean
  enable_transition: boolean
  transition_time: number
  admin_transition_time: number
  transition_remember_choice?: boolean
  transition_show_description?: boolean
  transition_theme?: string
  transition_color?: string
  transition_ad1?: string
  transition_ad2?: string
  announcement_enabled: boolean
  announcement_title?: string
  announcement_content?: string
  announcement_start?: string
  announcement_end?: string
  announcement_remember_days?: number
}

export interface HomeData {
  categories: Category[]
  featured: Website[]
  settings: PublicSettings
  user: User | null
}
