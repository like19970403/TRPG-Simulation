import { useAuthStore } from '../stores/auth-store'

const CARDS = [
  { icon: '📖', title: 'Scenarios', subtitle: 'Coming soon' },
  { icon: '🎲', title: 'Sessions', subtitle: 'Coming soon' },
  { icon: '⚔️', title: 'Characters', subtitle: 'Coming soon' },
]

export function DashboardPage() {
  const user = useAuthStore((s) => s.user)

  return (
    <div className="flex flex-col items-center px-6 pt-24">
      <h1 className="font-display text-5xl font-semibold text-text-primary">
        Welcome, {user?.username}
      </h1>
      <p className="mt-3 text-text-secondary">
        Your adventure awaits. More features coming soon.
      </p>

      <div className="mt-16 flex gap-6">
        {CARDS.map((card) => (
          <div
            key={card.title}
            className="flex h-[120px] w-[200px] flex-col items-center justify-center gap-2 rounded-lg border border-border bg-bg-card"
          >
            <span className="text-2xl">{card.icon}</span>
            <span className="text-sm font-medium text-text-primary">
              {card.title}
            </span>
            <span className="text-xs text-text-tertiary">{card.subtitle}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
