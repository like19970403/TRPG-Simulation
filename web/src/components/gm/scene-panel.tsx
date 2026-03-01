import { useGameStore } from '../../stores/game-store'
import { Button } from '../ui/button'

interface ScenePanelProps {
  sendAction: (type: string, payload: unknown) => void
}

export function ScenePanel({ sendAction }: ScenePanelProps) {
  const currentSceneId = useGameStore((s) => s.gameState?.current_scene)
  const scenarioContent = useGameStore((s) => s.scenarioContent)

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
          <div className="flex flex-wrap gap-2">
            {transitions.map((t, i) => (
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
              </Button>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
