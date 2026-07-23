/**
 * Fetch 封装：统一信封、Cookie Session、CSRF。
 */

export interface ApiErrorBody {
  code: string
  message: string
}

export interface ApiEnvelope<T> {
  success: boolean
  data?: T
  message?: string
  error?: ApiErrorBody
}

export class ApiError extends Error {
  status: number
  code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

const base = (import.meta.env.VITE_API_BASE || '').replace(/\/$/, '')

let csrfToken = ''

export function setCsrfToken(token: string) {
  csrfToken = token || ''
}

export function getCsrfToken() {
  return csrfToken
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers || {})
  if (!headers.has('Accept')) headers.set('Accept', 'application/json')
  const method = (init.method || 'GET').toUpperCase()
  if (method !== 'GET' && method !== 'HEAD' && csrfToken) {
    headers.set('X-CSRF-Token', csrfToken)
  }
  if (init.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  const res = await fetch(`${base}${path}`, {
    credentials: 'include',
    ...init,
    headers,
  })

  const body = (await res.json().catch(() => null)) as ApiEnvelope<T> | null

  if (!res.ok || !body?.success) {
    throw new ApiError(
      res.status,
      body?.error?.code || 'HTTP_ERROR',
      body?.error?.message || res.statusText || 'request failed',
    )
  }
  return body.data as T
}

export function apiGet<T>(path: string) {
  return request<T>(path, { method: 'GET' })
}

export function apiPost<T>(path: string, data?: unknown) {
  return request<T>(path, {
    method: 'POST',
    body: data === undefined ? undefined : JSON.stringify(data),
  })
}

export function apiPatch<T>(path: string, data?: unknown) {
  return request<T>(path, {
    method: 'PATCH',
    body: data === undefined ? undefined : JSON.stringify(data),
  })
}

export function apiPut<T>(path: string, data?: unknown) {
  return request<T>(path, {
    method: 'PUT',
    body: data === undefined ? undefined : JSON.stringify(data),
  })
}

export function apiDelete<T>(path: string, data?: unknown) {
  return request<T>(path, {
    method: 'DELETE',
    body: data === undefined ? undefined : JSON.stringify(data),
  })
}
