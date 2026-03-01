import { useGameStore } from '../../stores/game-store'
import { Button } from '../ui/button'
import { DiceRoller } from '../ui/dice-roller'

interface SceneViewProps {
  sendAction: (type: string, payload: unknown) => void
}

const EMPTY_DICE: never[] = []

export function SceneView({ sendAction }: SceneViewProps) {
  const currentScene = useGameStore((s) => s.gameState?.current_scene)
  const scene = useGameStore((s) =>
    s.scenarioContent?.scenes.find((sc) => sc.id === currentScene),
  )
  const diceHistory = useGameStore(
    (s) => s.gameState?.dice_history ?? EMPTY_DICE,
  )

  if (!scene) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <p className="text-sm text-text-tertiary">等待場景中...</p>
      </div>
    )
  }

  // Only show player_choice transitions
  const playerChoices = (scene.transitions ?? [])
    .map((t, originalIndex) => ({ ...t, originalIndex }))
    .filter((t) => t.trigger === 'player_choice')

  // Show last 5 dice results
  const recentDice = diceHistory.slice(-5).reverse()

  return (
    <div className="w-full max-w-2xl">
      {/* Parchment scene card */}
      <div className="rounded-xl border border-gold/30 bg-parchment p-8">
        <h2 className="mb-4 font-display text-2xl font-bold text-text-primary">
          {scene.name}
        </h2>
        <p className="whitespace-pre-wrap text-sm leading-relaxed text-text-secondary">
          {scene.content}
        </p>

        {/* Player choices */}
        {playerChoices.length > 0 && (
          <div className="mt-6 flex flex-col gap-2">
            <h3 className="text-xs font-semibold uppercase tracking-wider text-text-tertiary">
              你的選擇
            </h3>
            {playerChoices.map((choice) => (
              <Button
                key={choice.originalIndex}
                variant="secondary"
                className="justify-start border-gold/30 text-left"
                onClick={() =>
                  sendAction('player_choice', {
                    transition_index: choice.originalIndex,
                  })
                }
              >
                {choice.label ?? `前往 ${choice.target}`}
              </Button>
            ))}
          </div>
        )}
      </div>

      {/* Inline dice roller */}
      <div className="mt-6 rounded-xl border border-border bg-bg-card p-4">
        <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-text-tertiary">
          骰子
        </h3>
        <DiceRoller sendAction={sendAction} />

        {/* Recent dice results */}
        {recentDice.length > 0 && (
          <div className="mt-3 flex flex-col gap-1">
            {recentDice.map((dr, i) => (
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
