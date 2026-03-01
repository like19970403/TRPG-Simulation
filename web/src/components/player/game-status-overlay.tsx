import { Link } from 'react-router'
import { useGameStore } from '../../stores/game-store'
import { ROUTES } from '../../lib/constants'

interface GameStatusOverlayProps {
  /** GM should not see the paused overlay — they control resume */
  isGm?: boolean
}

export function GameStatusOverlay({ isGm }: GameStatusOverlayProps) {
  const status = useGameStore((s) => s.gameState?.status)

  if (status === 'paused' && !isGm) {
    return (
      <div className="fixed inset-0 z-40 flex flex-col items-center justify-center bg-[#0F0F0FCC]">
        <h2 className="font-display text-3xl font-bold text-gold">
          遊戲已暫停
        </h2>
        <p className="mt-3 text-sm text-text-tertiary">
          等待 GM 繼續遊戲...
        </p>
      </div>
    )
  }

  if (status === 'completed') {
    return (
      <div className="fixed inset-0 z-40 flex flex-col items-center justify-center bg-[#0F0F0FCC]">
        <h2 className="font-display text-3xl font-bold text-gold">
          遊戲結束
        </h2>
        <p className="mt-3 text-sm text-text-tertiary">
          遊戲場次已結束。
        </p>
        <Link
          to={ROUTES.DASHBOARD}
          className="mt-6 rounded-lg bg-gold px-6 py-2 text-sm font-medium text-bg-page transition-colors hover:bg-gold/80"
        >
          回到儀表板
        </Link>
      </div>
    )
  }

  return null
}
