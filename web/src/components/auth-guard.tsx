import { useEffect, useState } from 'react'
import { Navigate, Outlet } from 'react-router'
import { useAuthStore } from '../stores/auth-store'
import { useAuth } from '../hooks/use-auth'
import { ROUTES } from '../lib/constants'
import { LoadingSpinner } from './ui/loading-spinner'

export function AuthGuard() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const hasHydrated = useAuthStore.persist.hasHydrated()
  const { tryRefresh } = useAuth()
  const [refreshDone, setRefreshDone] = useState(false)

  useEffect(() => {
    if (!hasHydrated || isAuthenticated) return

    tryRefresh().finally(() => setRefreshDone(true))
  }, [hasHydrated, isAuthenticated, tryRefresh])

  // Still waiting for hydration or refresh attempt
  const checking = !hasHydrated || (!isAuthenticated && !refreshDone)

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
