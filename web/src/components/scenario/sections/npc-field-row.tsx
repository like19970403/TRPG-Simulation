import { Input } from '../../ui/input'
import { Select } from '../../ui/select'
import type { NPCField } from '../../../api/types'

interface NpcFieldRowProps {
  field: NPCField
  onChange: (f: NPCField) => void
  onRemove: () => void
}

export function NpcFieldRow({ field, onChange, onRemove }: NpcFieldRowProps) {
  return (
    <div className="flex flex-col gap-2 rounded-lg border border-border bg-[#1A1A1A] p-3">
      <div className="flex gap-2">
        <label className="flex flex-1 flex-col gap-1">
          <span className="text-xs text-text-tertiary">Key</span>
          <Input
            value={field.key}
            onChange={(e) => onChange({ ...field, key: e.target.value })}
            placeholder="欄位 key"
          />
        </label>
        <label className="flex flex-1 flex-col gap-1">
          <span className="text-xs text-text-tertiary">Label</span>
          <Input
            value={field.label}
            onChange={(e) => onChange({ ...field, label: e.target.value })}
            placeholder="顯示名稱"
          />
        </label>
      </div>
      <div className="flex items-end gap-2">
        <label className="flex flex-1 flex-col gap-1">
          <span className="text-xs text-text-tertiary">Value</span>
          <Input
            value={field.value}
            onChange={(e) => onChange({ ...field, value: e.target.value })}
            placeholder="欄位值"
          />
        </label>
        <label className="flex flex-col gap-1">
          <span className="text-xs text-text-tertiary">可見性</span>
          <Select
            value={field.visibility}
            onChange={(e) =>
              onChange({ ...field, visibility: e.target.value })
            }
            className="w-36"
          >
            <option value="visible">visible</option>
            <option value="hidden">hidden</option>
            <option value="gm_only">gm_only</option>
          </Select>
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
