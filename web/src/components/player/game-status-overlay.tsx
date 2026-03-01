import { Link } from 'react-router'
import { useGameStore } from '../../stores/game-store'
import { ROUTES } from '../../lib/constants'

export function GameStatusOverlay() {
  const status = useGameStore((s) => s.gameState?.status)

  if (status === 'paused') {
    return (
      <div className="fixed inset-0 z-40 flex flex-col items-center justify-center bg-[#0F0F0FCC]">
        <h2 className="font-display text-3xl font-bold text-gold">
          Game Paused
        </h2>
        <p className="mt-3 text-sm text-text-tertiary">
          Waiting for GM to resume...
        </p>
      </div>
    )
  }

  if (status === 'completed') {
    return (
      <div className="fixed inset-0 z-40 flex flex-col items-center justify-center bg-[#0F0F0FCC]">
        <h2 className="font-display text-3xl font-bold text-gold">
          Game Over
        </h2>
        <p className="mt-3 text-sm text-text-tertiary">
          The game session has ended.
        </p>
        <Link
          to={ROUTES.DASHBOARD}
          className="mt-6 rounded-lg bg-gold px-6 py-2 text-sm font-medium text-bg-page transition-colors hover:bg-gold/80"
        >
          Return to Dashboard
        </Link>
      </div>
    )
  }

  return null
}
