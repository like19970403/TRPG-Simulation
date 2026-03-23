import { useState } from 'react'
import { Input } from '../../ui/input'
import { Select } from '../../ui/select'
import { Button } from '../../ui/button'
import { ImageUpload } from '../../ui/image-upload'
import { NpcFieldRow } from './npc-field-row'
import type { NPC, NPCField, Item } from '../../../api/types'

interface NpcCardProps {
  npc: NPC
  onChange: (n: NPC) => void
  onRemove: () => void
  defaultExpanded?: boolean
  system?: string
  allItems?: Item[]
}

export function NpcCard({
  npc,
  onChange,
  onRemove,
  defaultExpanded = false,
  system,
  allItems = [],
}: NpcCardProps) {
  const [expanded, setExpanded] = useState(defaultExpanded)

  const isWuxia = system === 'wuxia'
  const attrs = npc.attributes ?? {}
  const wuxiaAttrs = ['武功', '內力', '身法', '機智']

  const weapons = allItems.filter((i) => i.slot === 'weapon')
  const armors = allItems.filter((i) => i.slot && i.slot !== 'weapon')
  const martialSkills = allItems.filter((i) => i.type === 'martial_skill')
  const cultivations = allItems.filter((i) => i.type === 'cultivation_method')

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

  const updateAttr = (key: string, val: number) => {
    onChange({ ...npc, attributes: { ...attrs, [key]: val } })
  }

  const toggleEquipment = (itemId: string) => {
    const eq = npc.equipment ?? []
    if (eq.includes(itemId)) {
      onChange({ ...npc, equipment: eq.filter((id) => id !== itemId) })
    } else {
      onChange({ ...npc, equipment: [...eq, itemId] })
    }
  }

  const toggleSkill = (itemId: string) => {
    const sk = npc.skills ?? []
    if (sk.includes(itemId)) {
      onChange({ ...npc, skills: sk.filter((id) => id !== itemId) })
    } else {
      onChange({ ...npc, skills: [...sk, itemId] })
    }
  }

  const hasCombatData = isWuxia && Object.keys(attrs).length > 0

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
            {npc.name || '（未命名）'}
          </span>
          {hasCombatData && (
            <span className="rounded bg-gold/20 px-1.5 py-0.5 text-[9px] text-gold">
              戰鬥
            </span>
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
          <label className="flex flex-col gap-1">
            <span className="text-xs font-medium text-text-secondary">
              名稱
            </span>
            <Input
              value={npc.name}
              onChange={(e) => onChange({ ...npc, name: e.target.value })}
              placeholder="NPC 名稱"
            />
          </label>

          <ImageUpload
            value={npc.image}
            onChange={(url) => onChange({ ...npc, image: url })}
            label="NPC 頭像（選填）"
          />

          {/* Combat Attributes (wuxia only) */}
          {isWuxia && (
            <div className="flex flex-col gap-3 rounded border border-gold/30 bg-gold/5 p-3">
              <span className="text-xs font-semibold text-gold">
                戰鬥屬性
              </span>

              {/* Base attributes */}
              <div className="grid grid-cols-4 gap-2">
                {wuxiaAttrs.map((attr) => (
                  <label key={attr} className="flex flex-col gap-1">
                    <span className="text-[10px] text-text-tertiary">{attr}</span>
                    <Input
                      type="number"
                      value={String(attrs[attr] ?? 5)}
                      onChange={(e) => updateAttr(attr, parseInt(e.target.value, 10) || 0)}
                      className="w-full"
                    />
                  </label>
                ))}
              </div>

              {/* HP */}
              <div className="flex items-center gap-3">
                <label className="flex flex-col gap-1">
                  <span className="text-[10px] text-text-tertiary">HP</span>
                  <Input
                    type="number"
                    value={String(npc.hp ?? (10 + (attrs['內力'] ?? 5) * 2))}
                    onChange={(e) => onChange({ ...npc, hp: parseInt(e.target.value, 10) || 20 })}
                    className="w-20"
                  />
                </label>
                <span className="mt-4 text-[9px] text-text-tertiary">
                  建議: 10 + 內力({attrs['內力'] ?? 5}) × 2 = {10 + (attrs['內力'] ?? 5) * 2}
                </span>
              </div>

              {/* Equipment */}
              <div className="flex flex-col gap-1">
                <span className="text-[10px] font-medium text-text-tertiary">裝備</span>
                <div className="flex flex-wrap gap-1.5">
                  {[...weapons, ...armors].map((item) => {
                    const selected = (npc.equipment ?? []).includes(item.id)
                    return (
                      <button
                        key={item.id}
                        type="button"
                        onClick={() => toggleEquipment(item.id)}
                        className={`rounded px-2 py-1 text-[10px] transition-colors ${
                          selected
                            ? 'bg-gold/20 text-gold'
                            : 'bg-border text-text-tertiary hover:text-text-secondary'
                        }`}
                      >
                        {item.name}
                        {item.atk ? ` atk${item.atk}` : ''}
                        {item.def ? ` def${item.def}` : ''}
                      </button>
                    )
                  })}
                  {weapons.length === 0 && armors.length === 0 && (
                    <span className="text-[9px] text-text-tertiary">請先在「道具」tab 新增武器/防具</span>
                  )}
                </div>
              </div>

              {/* Martial Skills */}
              <div className="flex flex-col gap-1">
                <span className="text-[10px] font-medium text-text-tertiary">武學</span>
                <div className="flex flex-wrap gap-1.5">
                  {martialSkills.map((item) => {
                    const selected = (npc.skills ?? []).includes(item.id)
                    return (
                      <button
                        key={item.id}
                        type="button"
                        onClick={() => toggleSkill(item.id)}
                        className={`rounded px-2 py-1 text-[10px] transition-colors ${
                          selected
                            ? 'bg-emerald-900/40 text-emerald-400'
                            : 'bg-border text-text-tertiary hover:text-text-secondary'
                        }`}
                      >
                        {item.name}
                      </button>
                    )
                  })}
                  {martialSkills.length === 0 && (
                    <span className="text-[9px] text-text-tertiary">無武學道具</span>
                  )}
                </div>
              </div>

              {/* Cultivation */}
              <label className="flex flex-col gap-1">
                <span className="text-[10px] font-medium text-text-tertiary">心法</span>
                <Select
                  value={npc.cultivation ?? ''}
                  onChange={(e) => onChange({ ...npc, cultivation: e.target.value || undefined })}
                  className="w-48"
                >
                  <option value="">— 無 —</option>
                  {cultivations.map((item) => (
                    <option key={item.id} value={item.id}>{item.name}</option>
                  ))}
                </Select>
              </label>
            </div>
          )}

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
