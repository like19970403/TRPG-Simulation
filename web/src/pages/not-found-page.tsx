import { Link } from 'react-router'
import { ROUTES } from '../lib/constants'

export function NotFoundPage() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-bg-page px-4">
      <h1 className="font-display text-6xl font-bold text-gold">404</h1>
      <p className="mt-4 text-text-secondary">Page not found</p>
      <Link
        to={ROUTES.HOME}
        className="mt-6 text-sm text-gold hover:text-gold-light"
      >
        Return home
      </Link>
    </div>
  )
}
