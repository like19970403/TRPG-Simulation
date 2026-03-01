import { Link, Outlet, useNavigate } from 'react-router'
import { useAuth } from '../hooks/use-auth'
import { ROUTES } from '../lib/constants'

export function AppLayout() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    navigate(ROUTES.LOGIN)
  }

  return (
    <div className="min-h-screen bg-bg-page">
      <nav className="flex h-14 items-center justify-between border-b border-border bg-bg-sidebar px-6">
        <div className="flex items-center gap-6">
          <Link to={ROUTES.DASHBOARD} className="flex items-center gap-2">
            <span className="text-lg">📜</span>
            <span className="font-display text-lg font-semibold text-gold">
              TRPG
            </span>
          </Link>
          <Link
            to={ROUTES.SCENARIOS}
            className="text-sm text-text-secondary transition-colors hover:text-text-primary"
          >
            Scenarios
          </Link>
          <Link
            to={ROUTES.SESSIONS}
            className="text-sm text-text-secondary transition-colors hover:text-text-primary"
          >
            Sessions
          </Link>
          <Link
            to={ROUTES.CHARACTERS}
            className="text-sm text-text-secondary transition-colors hover:text-text-primary"
          >
            Characters
          </Link>
        </div>
        <div className="flex items-center gap-4">
          <span className="text-sm text-text-secondary">
            {user?.username}
          </span>
          <button
            onClick={handleLogout}
            className="text-sm text-text-tertiary transition-colors hover:text-text-primary cursor-pointer"
          >
            Logout
          </button>
        </div>
      </nav>
      <main>
        <Outlet />
      </main>
    </div>
  )
}
