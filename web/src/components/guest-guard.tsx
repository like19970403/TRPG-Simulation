import { Navigate, Outlet } from 'react-router'
import { useAuthStore } from '../stores/auth-store'
import { ROUTES } from '../lib/constants'

export function GuestGuard() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)

  if (isAuthenticated) {
    return <Navigate to={ROUTES.DASHBOARD} replace />
  }

  return <Outlet />
}
