import { useState, useMemo } from 'react'
import { useGameStore } from '../../stores/game-store'
import { Input } from '../ui/input'
import { Select } from '../ui/select'
import type { NPC, Item } from '../../api/types'

interface CombatPanelProps {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  sendAction: (type: any, payload: any) => void
}

export function CombatPanel({ sendAction }: CombatPanelProps) {
  const variables = useGameStore((s) => s.gameState?.variables ?? {})
  const scenarioContent = useGameStore((s) => s.scenarioContent)
  const combatActive = variables.combat_status === 'active'

  const allNpcs = scenarioContent?.npcs ?? []
  const allItems = scenarioContent?.items ?? []

  // NPCs with combat attributes
  const combatNpcs = useMemo(
    () => allNpcs.filter((n: NPC) => n.attributes && Object.keys(n.attributes).length > 0),
    [allNpcs],
  )

  const [selectedNpcId, setSelectedNpcId] = useState('')
  const [enemyName, setEnemyName] = useState('')
  const [enemyHp, setEnemyHp] = useState('30')
  const [enemyMartial, setEnemyMartial] = useState('6')
  const [enemyDef, setEnemyDef] = useState('5')
  const [enemyAgility, setEnemyAgility] = useState('5')
  const [enemyWeaponAtk, setEnemyWeaponAtk] = useState('2')

  const handleSelectNpc = (npcId: string) => {
    setSelectedNpcId(npcId)
    if (!npcId) return

    const npc = allNpcs.find((n: NPC) => n.id === npcId)
    if (!npc?.attributes) return

    setEnemyName(npc.name)
    const attrs = npc.attributes
    const martial = attrs['武功'] ?? 5
    const innerForce = attrs['內力'] ?? 5
    const agility = attrs['身法'] ?? 5

    setEnemyMartial(String(martial))
    setEnemyAgility(String(agility))

    // Calculate HP
    const hp = npc.hp ?? 10 + innerForce * 2
    setEnemyHp(String(hp))

    // Calculate weapon atk from equipment
    const weaponAtk = (npc.equipment ?? [])
      .map((id) => allItems.find((i: Item) => i.id === id))
      .filter((i): i is Item => !!i && i.slot === 'weapon')
      .reduce((sum, i) => sum + (i.atk ?? 0), 0)
    setEnemyWeaponAtk(String(weaponAtk))

    // Calculate total armor def
    const armorDef = (npc.equipment ?? [])
      .map((id) => allItems.find((i: Item) => i.id === id))
      .filter((i): i is Item => !!i && i.slot !== 'weapon' && !!i.def)
      .reduce((sum, i) => sum + (i.def ?? 0), 0)
    setEnemyDef(String(armorDef))
  }

  const handleStartCombat = () => {
    const hp = parseInt(enemyHp) || 30
    sendAction('set_variable', { name: 'combat_enemy_name', value: enemyName })
    sendAction('set_variable', { name: 'combat_enemy_hp', value: hp })
    sendAction('set_variable', { name: 'combat_enemy_max_hp', value: hp })
    sendAction('set_variable', { name: 'combat_enemy_martial', value: parseInt(enemyMartial) || 6 })
    sendAction('set_variable', { name: 'combat_enemy_def', value: parseInt(enemyDef) || 5 })
    sendAction('set_variable', { name: 'combat_enemy_agility', value: parseInt(enemyAgility) || 5 })
    sendAction('set_variable', { name: 'combat_enemy_weapon_atk', value: parseInt(enemyWeaponAtk) || 2 })
    sendAction('set_variable', { name: 'combat_round', value: 1 })
    if (selectedNpcId) {
      sendAction('set_variable', { name: 'combat_enemy_npc_id', value: selectedNpcId })
    }
    sendAction('set_variable', { name: 'combat_status', value: 'active' })
    sendAction('gm_broadcast', { content: `**戰鬥開始！** 敵人：${enemyName}（HP ${hp}）` })
  }

  if (combatActive) {
    return (
      <div className="flex items-center justify-center p-4">
        <span className="text-xs text-gold">戰鬥進行中 — 請在戰鬥視窗操作</span>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-3 overflow-y-auto p-3">
      {/* NPC selector */}
      {combatNpcs.length > 0 && (
        <label className="flex flex-col gap-1">
          <span className="text-[10px] text-text-tertiary">選擇 NPC 敵人</span>
          <Select
            value={selectedNpcId}
            onChange={(e) => handleSelectNpc(e.target.value)}
          >
            <option value="">— 手動輸入 —</option>
            {combatNpcs.map((npc: NPC) => (
              <option key={npc.id} value={npc.id}>
                {npc.name} (HP {npc.hp ?? '?'})
              </option>
            ))}
          </Select>
        </label>
      )}

      <div className="grid grid-cols-2 gap-2">
        <label className="flex flex-col gap-1">
          <span className="text-[10px] text-text-tertiary">敵人名稱</span>
          <Input value={enemyName} onChange={(e) => setEnemyName(e.target.value)} />
        </label>
        <label className="flex flex-col gap-1">
          <span className="text-[10px] text-text-tertiary">HP</span>
          <Input type="number" value={enemyHp} onChange={(e) => setEnemyHp(e.target.value)} />
        </label>
        <label className="flex flex-col gap-1">
          <span className="text-[10px] text-text-tertiary">武功</span>
          <Input type="number" value={enemyMartial} onChange={(e) => setEnemyMartial(e.target.value)} />
        </label>
        <label className="flex flex-col gap-1">
          <span className="text-[10px] text-text-tertiary">裝備防禦</span>
          <Input type="number" value={enemyDef} onChange={(e) => setEnemyDef(e.target.value)} />
        </label>
        <label className="flex flex-col gap-1">
          <span className="text-[10px] text-text-tertiary">身法</span>
          <Input type="number" value={enemyAgility} onChange={(e) => setEnemyAgility(e.target.value)} />
        </label>
        <label className="flex flex-col gap-1">
          <span className="text-[10px] text-text-tertiary">武器攻擊力</span>
          <Input type="number" value={enemyWeaponAtk} onChange={(e) => setEnemyWeaponAtk(e.target.value)} />
        </label>
      </div>

      {selectedNpcId && (
        <div className="flex flex-wrap gap-1 text-[9px] text-text-tertiary">
          {(() => {
            const npc = allNpcs.find((n: NPC) => n.id === selectedNpcId)
            if (!npc) return null
            const skillNames = (npc.skills ?? [])
              .map((id) => allItems.find((i: Item) => i.id === id)?.name)
              .filter(Boolean)
            const cultName = allItems.find((i: Item) => i.id === npc.cultivation)?.name
            return (
              <>
                {skillNames.map((name) => (
                  <span key={name} className="rounded bg-emerald-900/40 px-1.5 py-0.5 text-emerald-400">
                    {name}
                  </span>
                ))}
                {cultName && (
                  <span className="rounded bg-amber-900/40 px-1.5 py-0.5 text-amber-400">
                    {cultName}
                  </span>
                )}
              </>
            )
          })()}
        </div>
      )}

      <button
        type="button"
        onClick={handleStartCombat}
        disabled={!enemyName.trim()}
        className="w-full bg-gold py-2.5 text-sm font-semibold text-bg-page disabled:opacity-40"
      >
        開始戰鬥
      </button>
    </div>
  )
}
