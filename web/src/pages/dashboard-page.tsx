import { Link } from 'react-router'
import { useAuthStore } from '../stores/auth-store'
import { ROUTES } from '../lib/constants'

const CARDS = [
  {
    title: 'Scenarios',
    subtitle: 'Create and manage your TRPG scenarios',
    to: ROUTES.SCENARIOS,
  },
  {
    title: 'Sessions',
    subtitle: 'Host or join game sessions',
    to: ROUTES.SESSIONS,
  },
]

export function DashboardPage() {
  const user = useAuthStore((s) => s.user)

  return (
    <div className="flex flex-col items-center px-6 pt-24">
      <h1 className="font-display text-5xl font-semibold text-text-primary">
        Welcome, {user?.username}
      </h1>
      <p className="mt-3 text-text-secondary">
        Your adventure awaits.
      </p>

      <div className="mt-16 flex gap-6">
        {CARDS.map((card) => (
          <Link
            key={card.title}
            to={card.to}
            className="flex h-35 w-60 flex-col items-center justify-center gap-3 rounded-lg border border-border bg-bg-card transition-colors hover:border-gold/40"
          >
            <span className="text-lg font-medium text-text-primary">
              {card.title}
            </span>
            <span className="text-center text-xs text-text-tertiary">
              {card.subtitle}
            </span>
          </Link>
        ))}
      </div>
    </div>
  )
}
