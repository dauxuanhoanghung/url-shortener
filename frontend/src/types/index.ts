export interface User {
  id: string
  email: string
  plan_code: string
  email_verified: boolean
  role: 'user' | 'admin'
}

export interface AuthResponse {
  access_token: string
  refresh_token: string
  user: User
}

export interface ApiResponse<T> {
  success: boolean
  data?: T
  error?: {
    code: string
    message: string
  }
}

export interface URLMetadata {
  title?: string | null
  description?: string | null
  og_image?: string | null
  favicon_url?: string | null
  fetch_status: 'pending' | 'ok' | 'failed'
}

export interface ShortURL {
  id: string
  short_code: string
  short_url: string
  original_url: string
  click_count: number
  created_at: string
  last_accessed_at?: string | null
  metadata: URLMetadata | null
}

export interface SSEUrlDeletedEvent {
  url_id: string
  short_code: string
  reason: string
  http_status: number
}

export interface ListURLResponse {
  urls: ShortURL[]
  total: number
}

export interface Plan {
  code: string
  name: string
  price_cents: number
  max_urls: number
  max_domains: number
  max_team_members: number
  analytics_retention_days: number
  api_rate_limit_per_min: number | null
  features: Record<string, boolean>
}
