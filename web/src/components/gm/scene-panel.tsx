import { useGameStore } from '../../stores/game-store'
import { Button } from '../ui/button'
import type { SendAction } from '../../hooks/use-game-socket'

interface ScenePanelProps {
  sendAction: SendAction
}

export function ScenePanel({ sendAction }: ScenePanelProps) {
  const currentSceneId = useGameStore((s) => s.gameState?.current_scene)
  const scenarioContent = useGameStore((s) => s.scenarioContent)
  const currentVotes = useGameStore((s) => s.currentVotes)

  // Resolve current scene from scenario content
  const scene = scenarioContent?.scenes?.find((s) => s.id === currentSceneId)

  if (!scene) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <p className="text-text-tertiary">
          {currentSceneId
            ? `場景「${currentSceneId}」未在劇本中找到`
            : '無活動場景'}
        </p>
      </div>
    )
  }

  const transitions = scene.transitions ?? []

  return (
    <div className="flex flex-1 flex-col gap-5 overflow-y-auto p-6">
      {/* Scene header */}
      <div className="flex items-center gap-3">
        <h2 className="font-display text-xl font-semibold text-text-primary">
          {scene.name}
        </h2>
        <span className="rounded bg-bg-input px-2 py-0.5 text-xs text-text-tertiary">
          {scene.id}
        </span>
      </div>

      {/* Scene content */}
      <div className="rounded-xl bg-card p-6">
        <p className="whitespace-pre-wrap text-sm leading-relaxed text-text-secondary">
          {scene.content}
        </p>
      </div>

      {/* GM Notes */}
      {scene.gm_notes && (
        <div className="rounded-xl border border-gold/30 bg-parchment p-6">
          <h3 className="mb-2 font-display text-sm font-semibold text-gold">
            GM 筆記
          </h3>
          <p className="whitespace-pre-wrap text-sm leading-relaxed text-text-secondary">
            {scene.gm_notes}
          </p>
        </div>
      )}

      {/* Scene Transitions */}
      <div>
        <h3 className="mb-3 text-sm font-semibold text-text-secondary">
          場景轉換
        </h3>
        {transitions.length === 0 ? (
          <p className="text-xs text-text-tertiary">
            劇本結束 — 無可用轉換
          </p>
        ) : (
          <div className="flex flex-col gap-3">
            <div className="flex flex-wrap gap-2">
              {transitions.map((t, i) => {
                const tally = currentVotes[String(i)]
                return (
                  <Button
                    key={`${t.target}-${i}`}
                    variant="secondary"
                    size="sm"
                    onClick={() =>
                      sendAction('advance_scene', { scene_id: t.target })
                    }
                  >
                    {t.label || t.target}
                    {t.trigger === 'auto' && (
                      <span className="ml-1 text-text-tertiary">(自動)</span>
                    )}
                    {t.trigger === 'condition_met' && (
                      <span className="ml-1 text-text-tertiary">(條件)</span>
                    )}
                    {t.trigger === 'player_choice' &&
                      tally &&
                      tally.count > 0 && (
                        <span className="ml-2 rounded-full bg-gold/20 px-2 py-0.5 text-xs font-medium text-gold">
                          {tally.count} 票
                        </span>
                      )}
                  </Button>
                )
              })}
            </div>

            {/* Voter details */}
            {Object.keys(currentVotes).length > 0 && (
              <div className="rounded-lg bg-bg-input p-3">
                <h4 className="mb-2 text-xs font-semibold text-text-tertiary">
                  投票詳情
                </h4>
                <div className="flex flex-col gap-1">
                  {Object.entries(currentVotes).map(([idx, tally]) => {
                    const t = transitions[Number(idx)]
                    return (
                      <div
                        key={idx}
                        className="flex items-center gap-2 text-xs text-text-secondary"
                      >
                        <span className="font-medium text-gold">
                          {t?.label || t?.target || `#${idx}`}
                        </span>
                        <span className="text-text-tertiary">—</span>
                        <span>{tally.voters.join(', ')}</span>
                      </div>
                    )
                  })}
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
