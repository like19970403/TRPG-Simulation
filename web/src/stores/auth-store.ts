import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { User } from '../api/types'

interface AuthState {
  accessToken: string | null
  user: User | null
  isAuthenticated: boolean
  setAuth: (accessToken: string) => void
  clearAuth: () => void
}

function parseJwtPayload(token: string): User | null {
  try {
    const base64Url = token.split('.')[1]
    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/')
    const payload = JSON.parse(atob(base64))
    return { id: payload.sub, username: payload.username }
  } catch {
    return null
  }
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      accessToken: null,
      user: null,
      isAuthenticated: false,

      setAuth: (accessToken: string) => {
        const user = parseJwtPayload(accessToken)
        set({ accessToken, user, isAuthenticated: !!user })
      },

      clearAuth: () => {
        set({ accessToken: null, user: null, isAuthenticated: false })
      },
    }),
    {
      name: 'trpg-auth',
      partialize: (state) => ({ accessToken: state.accessToken }),
      onRehydrateStorage: () => (state) => {
        if (state?.accessToken) {
          const user = parseJwtPayload(state.accessToken)
          state.user = user
          state.isAuthenticated = !!user
        }
      },
    },
  ),
)
