import { useGameStore } from '../../stores/game-store'
import { cn } from '../../lib/cn'

export function PlayerPanel() {
  const players = useGameStore((s) => s.gameState?.players)

  const playerList = players ? Object.entries(players) : []

  return (
    <div className="flex w-[260px] flex-col bg-bg-sidebar p-5">
      <div className="mb-4 flex items-center justify-between">
        <h2 className="font-display text-sm font-semibold uppercase tracking-wider text-gold">
          Players
        </h2>
        <span className="rounded-full bg-gold-tint px-2 py-0.5 text-xs font-medium text-gold">
          {playerList.length}
        </span>
      </div>

      {playerList.length === 0 ? (
        <p className="text-xs text-text-tertiary">No players connected</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {playerList.map(([playerId, player]) => (
            <li
              key={playerId}
              className="flex items-center gap-2 rounded-lg px-3 py-2 hover:bg-bg-input"
            >
              <span
                className={cn(
                  'h-2 w-2 rounded-full',
                  player.current_scene ? 'bg-success' : 'bg-text-tertiary',
                )}
              />
              <span className="text-sm text-text-primary">{playerId}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
