import { describe, it, expect, beforeEach, vi } from 'vitest'
import { apiClient, ApiClientError } from './client'
import { useAuthStore } from '../stores/auth-store'

// Create a valid JWT
function makeToken(sub: string, username: string): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const payload = btoa(JSON.stringify({ sub, username }))
  return `${header}.${payload}.fake-signature`
}

describe('apiClient', () => {
  beforeEach(() => {
    useAuthStore.getState().clearAuth()
    vi.restoreAllMocks()
  })

  it('attaches Authorization header when token is set', async () => {
    const token = makeToken('u1', 'gandalf')
    useAuthStore.getState().setAuth(token)

    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ data: 'ok' }),
    })
    vi.stubGlobal('fetch', mockFetch)

    await apiClient('/api/v1/test')

    const headers = mockFetch.mock.calls[0][1].headers as Headers
    expect(headers.get('Authorization')).toBe(`Bearer ${token}`)
  })

  it('does not attach Authorization header when no token', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ data: 'ok' }),
    })
    vi.stubGlobal('fetch', mockFetch)

    await apiClient('/api/v1/test')

    const headers = mockFetch.mock.calls[0][1].headers as Headers
    expect(headers.get('Authorization')).toBeNull()
  })

  it('sets Content-Type for requests with body', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({}),
    })
    vi.stubGlobal('fetch', mockFetch)

    await apiClient('/api/v1/test', {
      method: 'POST',
      body: JSON.stringify({ key: 'value' }),
    })

    const headers = mockFetch.mock.calls[0][1].headers as Headers
    expect(headers.get('Content-Type')).toBe('application/json')
  })

  it('throws ApiClientError on non-ok response', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 400,
      json: () =>
        Promise.resolve({
          error: 'VALIDATION_ERROR',
          message: 'Bad request',
          details: [],
        }),
    })
    vi.stubGlobal('fetch', mockFetch)

    await expect(apiClient('/api/v1/test')).rejects.toThrow(ApiClientError)
  })

  it('returns undefined for 204 responses', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 204,
    })
    vi.stubGlobal('fetch', mockFetch)

    const result = await apiClient('/api/v1/test')
    expect(result).toBeUndefined()
  })

  it('attempts refresh on 401 when token is set', async () => {
    const token = makeToken('u1', 'gandalf')
    useAuthStore.getState().setAuth(token)

    const newToken = makeToken('u1', 'gandalf')
    let callCount = 0

    const mockFetch = vi.fn().mockImplementation((url: string) => {
      if (url === '/api/v1/auth/refresh') {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: () => Promise.resolve({ accessToken: newToken }),
        })
      }
      callCount++
      if (callCount === 1) {
        return Promise.resolve({
          ok: false,
          status: 401,
          json: () =>
            Promise.resolve({ error: 'UNAUTHORIZED', message: 'Expired' }),
        })
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ data: 'ok' }),
      })
    })
    vi.stubGlobal('fetch', mockFetch)

    const result = await apiClient('/api/v1/test')
    expect(result).toEqual({ data: 'ok' })
    // Should have called: 1 original (401) + 1 refresh + 1 retry
    expect(mockFetch).toHaveBeenCalledTimes(3)
  })

  it('redirects to login when refresh fails', async () => {
    const token = makeToken('u1', 'gandalf')
    useAuthStore.getState().setAuth(token)

    const mockFetch = vi.fn().mockImplementation((url: string) => {
      if (url === '/api/v1/auth/refresh') {
        return Promise.resolve({ ok: false, status: 401 })
      }
      return Promise.resolve({
        ok: false,
        status: 401,
        json: () =>
          Promise.resolve({ error: 'UNAUTHORIZED', message: 'Expired' }),
      })
    })
    vi.stubGlobal('fetch', mockFetch)

    // Mock window.location
    const locationMock = { href: '' }
    vi.stubGlobal('location', locationMock)

    await expect(apiClient('/api/v1/test')).rejects.toThrow(ApiClientError)
    expect(locationMock.href).toBe('/login')
    expect(useAuthStore.getState().isAuthenticated).toBe(false)
  })
})
