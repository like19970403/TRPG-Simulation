import { useEffect, useState } from 'react'
import { Navigate, Outlet } from 'react-router'
import { useAuthStore } from '../stores/auth-store'
import { useAuth } from '../hooks/use-auth'
import { ROUTES } from '../lib/constants'
import { LoadingSpinner } from './ui/loading-spinner'

export function AuthGuard() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const { tryRefresh } = useAuth()
  const [checking, setChecking] = useState(!isAuthenticated)

  useEffect(() => {
    if (!isAuthenticated) {
      tryRefresh().finally(() => setChecking(false))
    }
  }, [isAuthenticated, tryRefresh])

  if (checking) {
    return (
      <div className="flex h-screen items-center justify-center bg-bg-page">
        <LoadingSpinner className="h-8 w-8 text-gold" />
      </div>
    )
  }

  if (!isAuthenticated) {
    return <Navigate to={ROUTES.LOGIN} replace />
  }

  return <Outlet />
}
