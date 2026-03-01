import { useEffect, useState } from 'react'
import { Navigate, Outlet, useParams } from 'react-router'
import { useAuthStore } from '../../stores/auth-store'
import { getSession } from '../../api/sessions'
import { LoadingSpinner } from '../ui/loading-spinner'

export function GmGuard() {
  const { id } = useParams<{ id: string }>()
  const user = useAuthStore((s) => s.user)
  const [loading, setLoading] = useState(() => !!(id && user))
  const [isGm, setIsGm] = useState(false)

  useEffect(() => {
    if (!id || !user) return

    let cancelled = false
    getSession(id)
      .then((session) => {
        if (!cancelled) setIsGm(session.gmId === user.id)
      })
      .catch(() => {
        if (!cancelled) setIsGm(false)
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [id, user])

  if (loading) {
    return (
      <div className="flex h-screen items-center justify-center bg-bg-page">
        <LoadingSpinner className="h-8 w-8 text-gold" />
      </div>
    )
  }

  if (!isGm) {
    return <Navigate to={`/sessions/${id}`} replace />
  }

  return <Outlet />
}
