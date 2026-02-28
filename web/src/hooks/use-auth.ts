import { useCallback } from 'react'
import { useAuthStore } from '../stores/auth-store'
import * as authApi from '../api/auth'
import type { RegisterRequest, LoginRequest } from '../api/types'

export function useAuth() {
  const { user, isAuthenticated, setAuth, clearAuth } = useAuthStore()

  const login = useCallback(
    async (data: LoginRequest) => {
      const res = await authApi.login(data)
      setAuth(res.accessToken)
    },
    [setAuth],
  )

  const register = useCallback(async (data: RegisterRequest) => {
    return authApi.register(data)
  }, [])

  const logout = useCallback(async () => {
    try {
      await authApi.logout()
    } finally {
      clearAuth()
    }
  }, [clearAuth])

  const tryRefresh = useCallback(async () => {
    try {
      const res = await authApi.refresh()
      setAuth(res.accessToken)
      return true
    } catch {
      clearAuth()
      return false
    }
  }, [setAuth, clearAuth])

  return { user, isAuthenticated, login, register, logout, tryRefresh }
}
