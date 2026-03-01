import { Input } from '../../ui/input'
import { Select } from '../../ui/select'
import type { Transition } from '../../../api/types'

interface TransitionEditorProps {
  transition: Transition
  onChange: (t: Transition) => void
  onRemove: () => void
  allSceneIds: string[]
  currentSceneId: string
}

export function TransitionEditor({
  transition,
  onChange,
  onRemove,
  allSceneIds,
  currentSceneId,
}: TransitionEditorProps) {
  const targetOptions = allSceneIds.filter((id) => id !== currentSceneId)

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
                {id}
              </option>
            ))}
          </Select>
        </label>
        <label className="flex flex-1 flex-col gap-1">
          <span className="text-xs text-text-tertiary">觸發方式</span>
          <Select
            value={transition.trigger}
            onChange={(e) =>
              onChange({ ...transition, trigger: e.target.value })
            }
          >
            <option value="player_choice">player_choice</option>
            <option value="auto">auto</option>
            <option value="gm">gm</option>
          </Select>
        </label>
      </div>
      <div className="flex items-end gap-2">
        <label className="flex flex-1 flex-col gap-1">
          <span className="text-xs text-text-tertiary">條件（選填）</span>
          <Input
            value={transition.condition ?? ''}
            onChange={(e) =>
              onChange({
                ...transition,
                condition: e.target.value || undefined,
              })
            }
            placeholder="expr 條件表達式"
          />
        </label>
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
