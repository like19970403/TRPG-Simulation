import { useGameStore } from '../../stores/game-store'
import { Button } from '../ui/button'
import { DiceRoller } from '../ui/dice-roller'
import { cn } from '../../lib/cn'

interface SceneViewProps {
  sendAction: (type: string, payload: unknown) => void
}

const EMPTY_DICE: never[] = []

export function SceneView({ sendAction }: SceneViewProps) {
  const currentScene = useGameStore((s) => s.gameState?.current_scene)
  const scene = useGameStore((s) =>
    s.scenarioContent?.scenes.find((sc) => sc.id === currentScene),
  )
  // Server-filtered transitions (conditions already evaluated per-player)
  const activeScene = useGameStore((s) => s.activeScene)
  const currentVotes = useGameStore((s) => s.currentVotes)
  const myVoteIndex = useGameStore((s) => s.myVoteIndex)
  const setMyVote = useGameStore((s) => s.setMyVote)
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

  // Use server-filtered transitions when available, fallback to client-side filtering
  const playerChoices =
    activeScene && activeScene.id === currentScene
      ? activeScene.transitions
      : (scene.transitions ?? [])
          .map((t, originalIndex) => ({
            ...t,
            transition_index: String(originalIndex),
          }))
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

        {/* Player vote buttons */}
        {playerChoices.length > 0 && (
          <div className="mt-6 flex flex-col gap-2">
            <h3 className="text-xs font-semibold uppercase tracking-wider text-text-tertiary">
              投票
            </h3>
            {playerChoices.map((choice) => {
              const idx = Number(choice.transition_index)
              const tally = currentVotes[choice.transition_index]
              const isMyVote = myVoteIndex === idx
              return (
                <Button
                  key={choice.transition_index}
                  variant="secondary"
                  className={cn(
                    'justify-start border-gold/30 text-left',
                    isMyVote && 'ring-2 ring-gold',
                  )}
                  onClick={() => {
                    setMyVote(idx)
                    sendAction('player_choice', {
                      transition_index: idx,
                    })
                  }}
                >
                  <span className="flex-1">
                    {choice.label ?? `前往 ${choice.target}`}
                  </span>
                  {tally && tally.count > 0 && (
                    <span className="ml-2 rounded-full bg-gold/20 px-2 py-0.5 text-xs text-gold">
                      {tally.count} 票
                    </span>
                  )}
                  {isMyVote && (
                    <span className="ml-1 text-xs text-gold">已投票</span>
                  )}
                </Button>
              )
            })}
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
