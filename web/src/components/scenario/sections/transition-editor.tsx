import { Input } from '../../ui/input'
import { Select } from '../../ui/select'
import { ConditionBuilder } from './condition-builder'
import type { Transition, Scene, ScenarioVariable, Item } from '../../../api/types'
import { TRIGGER_TYPE_LABELS } from '../../../lib/scenario-labels'

interface TransitionEditorProps {
  transition: Transition
  onChange: (t: Transition) => void
  onRemove: () => void
  allSceneIds: string[]
  allScenes: Scene[]
  currentSceneId: string
  allVariables: ScenarioVariable[]
  allItems?: Item[]
}

export function TransitionEditor({
  transition,
  onChange,
  onRemove,
  allSceneIds,
  allScenes,
  currentSceneId,
  allVariables,
  allItems = [],
}: TransitionEditorProps) {
  const targetOptions = allSceneIds.filter((id) => id !== currentSceneId)

  const sceneNameMap: Record<string, string> = {}
  for (const s of allScenes) {
    if (s.id) sceneNameMap[s.id] = s.name
  }

  return (
    <div className="flex flex-col gap-2 rounded-lg border border-border bg-[#1A1A1A] p-3">
      <div className="flex gap-2">
        <label className="flex flex-1 flex-col gap-1">
          <span className="text-xs text-text-tertiary">目標場景</span>
          <Select
            value={transition.target}
            onChange={(e) =>
              onChange({ ...transition, target: e.target.value })
            }
          >
            <option value="">-- 選擇場景 --</option>
            {targetOptions.map((id) => (
              <option key={id} value={id}>
                {sceneNameMap[id] || id}
              </option>
            ))}
          </Select>
        </label>
        <label className="flex flex-1 flex-col gap-1">
          <span className="text-xs text-text-tertiary">觸發方式</span>
          <Select
            value={transition.trigger}
            onChange={(e) => {
              const newTrigger = e.target.value
              onChange({
                ...transition,
                trigger: newTrigger,
                // Clear condition when switching to auto (it's ignored anyway)
                ...(newTrigger === 'auto' ? { condition: undefined } : {}),
              })
            }}
          >
            {Object.entries(TRIGGER_TYPE_LABELS).map(([val, label]) => (
              <option key={val} value={val}>
                {label}
              </option>
            ))}
          </Select>
        </label>
      </div>
      <div className="flex items-end gap-2">
        {transition.trigger !== 'auto' && (
          <div className="flex flex-1 flex-col gap-1">
            <span className="text-xs text-text-tertiary">條件（選填）</span>
            <ConditionBuilder
              value={transition.condition ?? ''}
              onChange={(expr) =>
                onChange({
                  ...transition,
                  condition: expr || undefined,
                })
              }
              allVariables={allVariables}
              allItems={allItems}
            />
          </div>
        )}
        <label className="flex flex-1 flex-col gap-1">
          <span className="text-xs text-text-tertiary">標籤（選填）</span>
          <Input
            value={transition.label ?? ''}
            onChange={(e) =>
              onChange({ ...transition, label: e.target.value || undefined })
            }
            placeholder="顯示給玩家的文字"
          />
        </label>
        <button
          type="button"
          onClick={onRemove}
          className="shrink-0 pb-2.5 text-sm text-text-tertiary transition-colors hover:text-error"
        >
          刪除
        </button>
      </div>
    </div>
  )
}
