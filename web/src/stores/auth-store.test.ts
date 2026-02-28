import { describe, it, expect, beforeEach } from 'vitest'
import { useAuthStore } from './auth-store'

// Create a valid JWT with sub and username claims
function makeToken(sub: string, username: string): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const payload = btoa(JSON.stringify({ sub, username }))
  return `${header}.${payload}.fake-signature`
}

describe('auth-store', () => {
  beforeEach(() => {
    useAuthStore.getState().clearAuth()
  })

  it('starts unauthenticated', () => {
    const state = useAuthStore.getState()
    expect(state.isAuthenticated).toBe(false)
    expect(state.accessToken).toBeNull()
    expect(state.user).toBeNull()
  })

  it('setAuth parses JWT and sets user', () => {
    const token = makeToken('user-123', 'aragorn')
    useAuthStore.getState().setAuth(token)

    const state = useAuthStore.getState()
    expect(state.isAuthenticated).toBe(true)
    expect(state.accessToken).toBe(token)
    expect(state.user).toEqual({ id: 'user-123', username: 'aragorn' })
  })

  it('clearAuth resets state', () => {
    const token = makeToken('user-123', 'aragorn')
    useAuthStore.getState().setAuth(token)
    useAuthStore.getState().clearAuth()

    const state = useAuthStore.getState()
    expect(state.isAuthenticated).toBe(false)
    expect(state.accessToken).toBeNull()
    expect(state.user).toBeNull()
  })

  it('setAuth with invalid token sets isAuthenticated to false', () => {
    useAuthStore.getState().setAuth('not-a-jwt')

    const state = useAuthStore.getState()
    expect(state.isAuthenticated).toBe(false)
    expect(state.user).toBeNull()
  })
})
