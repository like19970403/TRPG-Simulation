import { useGameStore } from '../../stores/game-store'
import { Markdown } from '../ui/markdown'

export function RulesReferencePanel() {
  const gmReference = useGameStore(
    (s) => s.scenarioContent?.rules?.gm_reference,
  )

  if (!gmReference) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <p className="text-xs text-text-tertiary">
          此劇本未定義規則參考。可在劇本編輯器的「規則」分頁中設定。
        </p>
      </div>
    )
  }

  return (
    <div className="overflow-y-auto p-4">
      <Markdown className="text-xs">{gmReference}</Markdown>
    </div>
  )
}
