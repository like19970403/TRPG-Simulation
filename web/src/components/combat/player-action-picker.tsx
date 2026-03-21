import { useState } from 'react'
import { cn } from '../../lib/cn'
import { parseSkillCost, parseSkillLevel } from '../../lib/combat-utils'
import type { Item, InventoryEntry } from '../../api/types'

export interface CombatAction {
  type: 'attack' | 'skill' | 'defend' | 'item' | 'flee'
  skillId?: string
  skillName?: string
  itemId?: string
  target?: string
}

interface PlayerActionPickerProps {
  inventory: InventoryEntry[]
  allItems: Item[]
  equippedWeaponType?: string
  innerForceCount: number
  onConfirm: (action: CombatAction) => void
  confirmed: boolean
  onModify: () => void
  confirmedAction?: CombatAction | null
}

export function PlayerActionPicker({
  inventory,
  allItems,
  equippedWeaponType: _equippedWeaponType,
  innerForceCount,
  onConfirm,
  confirmed,
  onModify,
  confirmedAction,
}: PlayerActionPickerProps) {
  const [selectedType, setSelectedType] = useState<CombatAction['type'] | null>(null)
  const [selectedSkillId, setSelectedSkillId] = useState('')

  // Show all martial skills from inventory (already filtered at character creation)
  const martialSkills = inventory
    .map((e) => allItems.find((i) => i.id === e.item_id))
    .filter((i): i is Item => !!i && i.type === 'martial_skill')

  // Consumable items
  const consumables = inventory
    .map((e) => {
      const item = allItems.find((i) => i.id === e.item_id)
      return item && item.type === 'consumable' ? { item, quantity: e.quantity } : null
    })
    .filter(Boolean) as { item: Item; quantity: number }[]

  const handleConfirm = () => {
    if (!selectedType) return
    const action: CombatAction = { type: selectedType, target: 'enemy' }
    if (selectedType === 'skill') {
      const skill = allItems.find((i) => i.id === selectedSkillId)
      action.skillId = selectedSkillId
      action.skillName = skill?.name
    }
    onConfirm(action)
  }

  if (confirmed && confirmedAction) {
    const label =
      confirmedAction.type === 'attack' ? '普通攻擊'
        : confirmedAction.type === 'skill' ? `武學：${confirmedAction.skillName}`
          : confirmedAction.type === 'defend' ? '防禦'
            : confirmedAction.type === 'item' ? '使用道具'
              : '逃跑'
    return (
      <div className="flex flex-col items-center gap-3 py-4">
        <span className="text-sm font-medium text-gold">✓ 已確定：{label}</span>
        <button
          type="button"
          onClick={onModify}
          className="text-[10px] text-text-tertiary underline hover:text-text-secondary"
        >
          修改行動
        </button>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-3">
      <span className="text-xs font-semibold text-text-secondary">選擇行動：</span>

      {/* Basic action buttons */}
      <div className="grid grid-cols-4 gap-2">
        {([
          { type: 'attack' as const, icon: '⚔', label: '攻擊', desc: '普通攻擊' },
          { type: 'defend' as const, icon: '🛡', label: '防禦', desc: '防禦翻倍' },
          { type: 'item' as const, icon: '🎒', label: '道具', desc: '使用道具' },
          { type: 'flee' as const, icon: '🏃', label: '逃跑', desc: '2d6+身法≥10' },
        ]).map((opt) => (
          <button
            key={opt.type}
            type="button"
            onClick={() => { setSelectedType(opt.type); setSelectedSkillId('') }}
            className={cn(
              'flex flex-col items-center gap-1 rounded-none border p-3 transition-colors',
              selectedType === opt.type
                ? 'border-gold bg-gold/10'
                : 'border-border bg-bg-sidebar hover:border-text-tertiary',
            )}
          >
            <span className="text-sm">{opt.icon}</span>
            <span className="text-xs font-medium text-text-primary">{opt.label}</span>
            <span className="text-[9px] text-text-tertiary">{opt.desc}</span>
          </button>
        ))}
      </div>

      {/* Martial skill cards */}
      {martialSkills.length > 0 && (
        <div className="grid grid-cols-2 gap-2">
          {martialSkills.map((skill) => {
            const cost = parseSkillCost(skill)
            const disabled = innerForceCount < cost
            return (
              <button
                key={skill.id}
                type="button"
                disabled={disabled}
                onClick={() => { setSelectedType('skill'); setSelectedSkillId(skill.id) }}
                className={cn(
                  'flex flex-col gap-1 rounded-none border p-2.5 text-left transition-colors',
                  selectedType === 'skill' && selectedSkillId === skill.id
                    ? 'border-gold bg-gold/10'
                    : disabled
                      ? 'cursor-not-allowed border-border opacity-40'
                      : 'border-border bg-bg-sidebar hover:border-text-tertiary',
                )}
              >
                <div className="flex items-center gap-2">
                  <span className="text-xs font-medium text-text-primary">{skill.name}</span>
                  <span className="rounded bg-emerald-900/40 px-1 py-0.5 text-[8px] text-emerald-400">{parseSkillLevel(skill)}</span>
                </div>
                <span className="text-[9px] text-text-tertiary">
                  消耗 {cost} 內力 {skill.description?.match(/武功[+＋]\d/)?.[0] ?? ''}
                </span>
              </button>
            )
          })}
        </div>
      )}

      {/* Item selection (when item type selected) */}
      {selectedType === 'item' && consumables.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {consumables.map(({ item, quantity }) => (
            <button
              key={item.id}
              type="button"
              onClick={() => { setSelectedSkillId(item.id) }}
              className={cn(
                'rounded-none border px-3 py-1.5 text-xs transition-colors',
                selectedSkillId === item.id
                  ? 'border-gold bg-gold/10 text-gold'
                  : 'border-border text-text-secondary hover:border-text-tertiary',
              )}
            >
              {item.name} ×{quantity}
            </button>
          ))}
        </div>
      )}

      {/* Confirm button */}
      <div className="flex justify-end">
        <button
          type="button"
          onClick={handleConfirm}
          disabled={!selectedType}
          className="rounded-none bg-gold px-7 py-2.5 text-[13px] font-semibold text-bg-page transition-colors disabled:opacity-40"
        >
          確定行動
        </button>
      </div>
    </div>
  )
}
