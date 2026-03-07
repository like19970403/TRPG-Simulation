import { useState } from 'react'
import { Link, Outlet, useNavigate } from 'react-router'
import { useAuth } from '../hooks/use-auth'
import { ROUTES } from '../lib/constants'
import { cn } from '../lib/cn'

export function AppLayout() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const [menuOpen, setMenuOpen] = useState(false)

  const handleLogout = async () => {
    await logout()
    navigate(ROUTES.LOGIN)
  }

  const navLinks = [
    { to: ROUTES.SCENARIOS, label: '劇本' },
    { to: ROUTES.SESSIONS, label: '場次' },
    { to: ROUTES.CHARACTERS, label: '角色' },
  ]

  return (
    <div className="min-h-screen bg-bg-page safe-top">
      <nav className="border-b border-border bg-bg-sidebar">
        <div className="flex h-14 items-center justify-between px-4 md:px-6">
          <div className="flex items-center gap-6">
            <Link to={ROUTES.DASHBOARD} className="flex items-center gap-2">
              <span className="text-lg">📜</span>
              <span className="font-display text-lg font-semibold text-gold">
                TRPG
              </span>
            </Link>
            {navLinks.map((link) => (
              <Link
                key={link.to}
                to={link.to}
                className="hidden text-sm text-text-secondary transition-colors hover:text-text-primary md:block"
              >
                {link.label}
              </Link>
            ))}
          </div>
          <div className="hidden items-center gap-4 md:flex">
            <span className="text-sm text-text-secondary">
              {user?.username}
            </span>
            <button
              onClick={handleLogout}
              className="cursor-pointer text-sm text-text-tertiary transition-colors hover:text-text-primary"
            >
              登出
            </button>
          </div>
          {/* Mobile hamburger */}
          <button
            className="cursor-pointer text-text-secondary md:hidden"
            onClick={() => setMenuOpen((v) => !v)}
            aria-label="選單"
          >
            <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              {menuOpen ? (
                <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
              ) : (
                <path strokeLinecap="round" strokeLinejoin="round" d="M4 6h16M4 12h16M4 18h16" />
              )}
            </svg>
          </button>
        </div>

        {/* Mobile dropdown */}
        <div className={cn('flex-col gap-1 border-t border-border px-4 py-3 md:hidden', menuOpen ? 'flex' : 'hidden')}>
          {navLinks.map((link) => (
            <Link
              key={link.to}
              to={link.to}
              className="rounded-md px-3 py-2 text-sm text-text-secondary transition-colors hover:bg-bg-input hover:text-text-primary"
              onClick={() => setMenuOpen(false)}
            >
              {link.label}
            </Link>
          ))}
          <div className="my-1 h-px bg-border" />
          <div className="flex items-center justify-between px-3 py-2">
            <span className="text-sm text-text-secondary">{user?.username}</span>
            <button
              onClick={handleLogout}
              className="cursor-pointer text-sm text-text-tertiary transition-colors hover:text-text-primary"
            >
              登出
            </button>
          </div>
        </div>
      </nav>
      <main>
        <Outlet />
      </main>
    </div>
  )
}
