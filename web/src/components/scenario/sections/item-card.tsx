import { useState } from 'react'
import { Input } from '../../ui/input'
import { Textarea } from '../../ui/textarea'
import { Select } from '../../ui/select'
import type { Item } from '../../../api/types'

interface ItemCardProps {
  item: Item
  onChange: (i: Item) => void
  onRemove: () => void
  defaultExpanded?: boolean
}

export function ItemCard({
  item,
  onChange,
  onRemove,
  defaultExpanded = false,
}: ItemCardProps) {
  const [expanded, setExpanded] = useState(defaultExpanded)

  return (
    <div className="rounded-lg border border-border bg-bg-card">
      {/* Header */}
      <div
        className="flex cursor-pointer items-center justify-between px-4 py-3"
        onClick={() => setExpanded(!expanded)}
      >
        <div className="flex items-center gap-2">
          <span className="text-xs text-text-tertiary">
            {expanded ? '▼' : '▶'}
          </span>
          <span className="text-sm font-medium text-text-primary">
            {item.id || '（未命名）'}
          </span>
          {item.name && (
            <span className="text-sm text-text-secondary">— {item.name}</span>
          )}
        </div>
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation()
            onRemove()
          }}
          className="text-xs text-text-tertiary transition-colors hover:text-error"
        >
          刪除
        </button>
      </div>

      {/* Body */}
      {expanded && (
        <div className="flex flex-col gap-4 border-t border-border px-4 py-4">
          <div className="flex gap-3">
            <label className="flex flex-1 flex-col gap-1">
              <span className="text-xs font-medium text-text-secondary">
                道具 ID
              </span>
              <Input
                value={item.id}
                onChange={(e) => onChange({ ...item, id: e.target.value })}
                placeholder="道具 ID"
              />
            </label>
            <label className="flex flex-1 flex-col gap-1">
              <span className="text-xs font-medium text-text-secondary">
                名稱
              </span>
              <Input
                value={item.name}
                onChange={(e) => onChange({ ...item, name: e.target.value })}
                placeholder="道具名稱"
              />
            </label>
            <label className="flex flex-col gap-1">
              <span className="text-xs font-medium text-text-secondary">
                類型
              </span>
              <Select
                value={item.type}
                onChange={(e) => onChange({ ...item, type: e.target.value })}
                className="w-36"
              >
                <option value="item">道具</option>
                <option value="clue">線索</option>
                <option value="consumable">消耗品</option>
              </Select>
            </label>
          </div>

          <label className="flex flex-col gap-1">
            <span className="text-xs font-medium text-text-secondary">
              描述
            </span>
            <Textarea
              value={item.description}
              onChange={(e) =>
                onChange({ ...item, description: e.target.value })
              }
              rows={3}
              placeholder="道具描述..."
            />
          </label>

          <label className="flex flex-col gap-1">
            <span className="text-xs font-medium text-text-secondary">
              圖片 URL（選填）
            </span>
            <Input
              value={item.image ?? ''}
              onChange={(e) =>
                onChange({ ...item, image: e.target.value || undefined })
              }
              placeholder="https://..."
            />
          </label>
        </div>
      )}
    </div>
  )
}
