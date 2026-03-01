import { useState } from 'react'
import { Input } from '../../ui/input'
import { Button } from '../../ui/button'
import { NpcFieldRow } from './npc-field-row'
import type { NPC, NPCField } from '../../../api/types'

interface NpcCardProps {
  npc: NPC
  onChange: (n: NPC) => void
  onRemove: () => void
  defaultExpanded?: boolean
}

export function NpcCard({
  npc,
  onChange,
  onRemove,
  defaultExpanded = false,
}: NpcCardProps) {
  const [expanded, setExpanded] = useState(defaultExpanded)

  const updateField = (index: number, f: NPCField) => {
    const fields = [...(npc.fields ?? [])]
    fields[index] = f
    onChange({ ...npc, fields })
  }

  const removeField = (index: number) => {
    const fields = (npc.fields ?? []).filter((_, i) => i !== index)
    onChange({ ...npc, fields })
  }

  const addField = () => {
    const fields = [
      ...(npc.fields ?? []),
      { key: '', label: '', value: '', visibility: 'hidden' },
    ]
    onChange({ ...npc, fields })
  }

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
            {npc.id || '（未命名）'}
          </span>
          {npc.name && (
            <span className="text-sm text-text-secondary">— {npc.name}</span>
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
                NPC ID
              </span>
              <Input
                value={npc.id}
                onChange={(e) => onChange({ ...npc, id: e.target.value })}
                placeholder="NPC ID"
              />
            </label>
            <label className="flex flex-1 flex-col gap-1">
              <span className="text-xs font-medium text-text-secondary">
                名稱
              </span>
              <Input
                value={npc.name}
                onChange={(e) => onChange({ ...npc, name: e.target.value })}
                placeholder="NPC 名稱"
              />
            </label>
          </div>

          <label className="flex flex-col gap-1">
            <span className="text-xs font-medium text-text-secondary">
              圖片 URL（選填）
            </span>
            <Input
              value={npc.image ?? ''}
              onChange={(e) =>
                onChange({ ...npc, image: e.target.value || undefined })
              }
              placeholder="https://..."
            />
          </label>

          {/* Fields */}
          <div className="flex flex-col gap-2">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium text-text-secondary">
                欄位資料
              </span>
              <Button
                variant="secondary"
                size="sm"
                onClick={addField}
                type="button"
              >
                + 新增欄位
              </Button>
            </div>
            {(npc.fields ?? []).length === 0 && (
              <p className="text-xs text-text-tertiary">無欄位</p>
            )}
            {(npc.fields ?? []).map((f, i) => (
              <NpcFieldRow
                key={i}
                field={f}
                onChange={(val) => updateField(i, val)}
                onRemove={() => removeField(i)}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
