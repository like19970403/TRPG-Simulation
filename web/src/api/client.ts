import type { ApiError } from './types'
import { useAuthStore } from '../stores/auth-store'
import { API, ROUTES } from '../lib/constants'

export class ApiClientError extends Error {
  status: number
  body: ApiError

  constructor(status: number, body: ApiError) {
    super(body.message)
    this.name = 'ApiClientError'
    this.status = status
    this.body = body
  }
}

let isRefreshing = false
let refreshPromise: Promise<boolean> | null = null

async function tryRefresh(): Promise<boolean> {
  try {
    const res = await fetch(API.REFRESH, {
      method: 'POST',
      credentials: 'include',
    })
    if (!res.ok) return false
    const data = await res.json()
    if (!data.accessToken || typeof data.accessToken !== 'string') return false
    useAuthStore.getState().setAuth(data.accessToken)
    return true
  } catch {
    return false
  }
}

export async function apiClient<T>(
  url: string,
  options: RequestInit = {},
): Promise<T> {
  const token = useAuthStore.getState().accessToken

  const headers = new Headers(options.headers)
  if (!headers.has('Content-Type') && options.body) {
    headers.set('Content-Type', 'application/json')
  }
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  let res = await fetch(url, {
    ...options,
    headers,
    credentials: 'include',
  })

  if (res.status === 401 && token) {
    if (!isRefreshing) {
      isRefreshing = true
      refreshPromise = tryRefresh().finally(() => {
        isRefreshing = false
        refreshPromise = null
      })
    }

    const refreshed = await refreshPromise
    if (refreshed) {
      const newToken = useAuthStore.getState().accessToken
      headers.set('Authorization', `Bearer ${newToken}`)
      res = await fetch(url, {
        ...options,
        headers,
        credentials: 'include',
      })
    } else {
      useAuthStore.getState().clearAuth()
      window.location.href = ROUTES.LOGIN
      throw new ApiClientError(401, {
        error: 'UNAUTHORIZED',
        message: 'Session expired',
      })
    }
  }

  if (!res.ok) {
    const body: ApiError = await res.json().catch(() => ({
      error: 'UNKNOWN',
      message: 'An unexpected error occurred',
    }))
    throw new ApiClientError(res.status, body)
  }

  if (res.status === 204) {
    return undefined as T
  }

  return res.json()
}
