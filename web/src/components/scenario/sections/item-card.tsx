import { useState } from 'react'
import { Input } from '../../ui/input'
import { Textarea } from '../../ui/textarea'
import { Select } from '../../ui/select'
import { ImageUpload } from '../../ui/image-upload'
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
            {item.name || '（未命名）'}
          </span>
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
              描述（玩家可見）
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
              GM 筆記（僅 GM 可見）
            </span>
            <Textarea
              value={item.gm_notes ?? ''}
              onChange={(e) =>
                onChange({
                  ...item,
                  gm_notes: e.target.value || undefined,
                })
              }
              rows={2}
              placeholder="GM 專屬筆記..."
            />
          </label>

          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={!!item.stackable}
              onChange={(e) =>
                onChange({
                  ...item,
                  stackable: e.target.checked || undefined,
                })
              }
              className="h-4 w-4 rounded border-border bg-bg-input accent-gold"
            />
            <span className="text-xs font-medium text-text-secondary">
              可堆疊（允許同一道具多個數量）
            </span>
          </label>

          <ImageUpload
            value={item.image}
            onChange={(url) => onChange({ ...item, image: url })}
            label="道具圖片（選填）"
          />
        </div>
      )}
    </div>
  )
}
