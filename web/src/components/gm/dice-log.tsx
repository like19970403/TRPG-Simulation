import { useGameStore } from '../../stores/game-store'
import { DiceRoller } from '../ui/dice-roller'
import type { SendAction } from '../../hooks/use-game-socket'

interface DiceLogProps {
  sendAction: SendAction
}

const EMPTY_DICE: never[] = []

export function DiceLog({ sendAction }: DiceLogProps) {
  const diceHistory = useGameStore(
    (s) => s.gameState?.dice_history ?? EMPTY_DICE,
  )

  return (
    <div className="flex flex-1 flex-col p-4">
      {/* Roll input with cache */}
      <div className="mb-3">
        <DiceRoller sendAction={sendAction} showPurpose />
      </div>

      {/* Dice history */}
      <div className="flex-1 overflow-y-auto">
        {diceHistory.length === 0 ? (
          <p className="text-xs text-text-tertiary">尚無骰子紀錄</p>
        ) : (
          <div className="flex flex-col gap-1">
            {[...diceHistory].reverse().map((dr, i) => (
              <div
                key={`dice-${i}`}
                className="flex flex-col gap-0.5 text-xs"
              >
                <div className="flex items-center gap-2">
                  {dr.roller_name && (
                    <span className="font-medium text-text-secondary">
                      {dr.roller_name}
                    </span>
                  )}
                  <span className="font-mono font-medium text-gold">
                    {dr.formula}
                  </span>
                  <span className="text-text-tertiary">
                    [{dr.results.join(', ')}]
                    {dr.modifier !== 0 &&
                      (dr.modifier > 0
                        ? `+${dr.modifier}`
                        : `${dr.modifier}`)}
                  </span>
                  <span className="font-medium text-text-primary">
                    = {dr.total}
                  </span>
                </div>
                {dr.purpose && (
                  <div className="pl-1 text-text-tertiary">
                    {dr.purpose}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
