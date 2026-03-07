import { Link } from 'react-router'
import { useAuthStore } from '../stores/auth-store'
import { ROUTES } from '../lib/constants'

const CARDS = [
  {
    title: '劇本',
    subtitle: '建立與管理你的 TRPG 劇本',
    to: ROUTES.SCENARIOS,
  },
  {
    title: '場次',
    subtitle: '主持或加入遊戲場次',
    to: ROUTES.SESSIONS,
  },
  {
    title: '角色',
    subtitle: '建立與管理你的角色',
    to: ROUTES.CHARACTERS,
  },
]

export function DashboardPage() {
  const user = useAuthStore((s) => s.user)

  return (
    <div className="flex flex-col items-center px-4 pt-12 md:px-6 md:pt-24">
      <h1 className="font-display text-3xl font-semibold text-text-primary md:text-5xl">
        歡迎，{user?.username}
      </h1>
      <p className="mt-3 text-text-secondary">
        你的冒險即將展開。
      </p>

      <div className="mt-10 flex w-full max-w-3xl flex-col gap-4 md:mt-16 md:flex-row md:gap-6">
        {CARDS.map((card) => (
          <Link
            key={card.title}
            to={card.to}
            className="flex h-28 flex-1 flex-col items-center justify-center gap-3 rounded-lg border border-border bg-bg-card transition-colors hover:border-gold/40 md:h-35"
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
