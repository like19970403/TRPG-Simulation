import type { CombatAction } from './player-action-picker'
import type { CombatLogEntry } from './combat-log'
import type { Item, InventoryEntry } from '../../api/types'

export interface Combatant {
  id: string
  name: string
  isEnemy: boolean
  martial: number
  innerForce: number
  agility: number
  wisdom: number
  weaponAtk: number
  weaponType?: string
  armorDef: number
  action: CombatAction
  hpVar: string
  currentHp: number
  maxHp: number
  cultivationBonus: number
  cultivationWeaponMatch: boolean
}

export interface RoundResult {
  logs: CombatLogEntry[]
  hpChanges: { varName: string; newValue: number }[]
  itemRemovals: { playerId: string; itemId: string; quantity: number }[]
  diceRolls: { purpose: string; formula: string }[]
}

let logCounter = 0
function logId() {
  return `log-${Date.now()}-${logCounter++}`
}

function parseCost(skill: Item | undefined): number {
  if (!skill?.gm_notes) return 2
  const match = skill.gm_notes.match(/消耗:\s*(\d+)/)
  return match ? parseInt(match[1]) : 2
}

function parseSkillBonus(skill: Item | undefined): number {
  if (!skill?.gm_notes) return 0
  const match = skill.gm_notes.match(/武功\s*[+＋]\s*(\d+)/)
  return match ? parseInt(match[1]) : 0
}

/**
 * Calculate the full round result without sending any actions.
 * The caller is responsible for dispatching the actions sequentially.
 */
export function calculateRound(
  combatants: Combatant[],
  allItems: Item[],
  _playerInventories: Record<string, InventoryEntry[]>,
): RoundResult {
  // Sort by agility (descending)
  const sorted = [...combatants].sort((a, b) => b.agility - a.agility)

  const logs: CombatLogEntry[] = []
  const hpChanges: { varName: string; newValue: number }[] = []
  const itemRemovals: { playerId: string; itemId: string; quantity: number }[] = []
  const diceRolls: { purpose: string; formula: string }[] = []

  // Track HP mutations within the round
  const hpState: Record<string, number> = {}
  for (const c of sorted) {
    hpState[c.id] = c.currentHp
  }

  // Mark who is defending this round
  const defending = new Set<string>()
  for (const c of sorted) {
    if (c.action.type === 'defend') defending.add(c.id)
  }

  for (const actor of sorted) {
    // Skip dead
    if (hpState[actor.id] <= 0) continue

    const action = actor.action

    if (action.type === 'defend') {
      logs.push({ id: logId(), text: `${actor.name} 選擇防禦，本回合防禦力翻倍`, type: 'info' })
      continue
    }

    if (action.type === 'flee') {
      // 2d6 + agility >= 10
      diceRolls.push({ purpose: `${actor.name} 嘗試逃跑`, formula: '2d6' })
      logs.push({ id: logId(), text: `${actor.name} 嘗試逃跑（2d6+身法${actor.agility}≥10）`, type: 'action' })
      // Actual dice result will be filled in by the executor
      continue
    }

    if (action.type === 'item') {
      logs.push({ id: logId(), text: `${actor.name} 使用道具`, type: 'info' })
      continue
    }

    // Attack or Skill
    const targetId = action.target ?? 'enemy'
    const target = sorted.find((c) => {
      if (targetId === 'enemy') return c.isEnemy
      return c.id === targetId
    })
    if (!target || hpState[target.id] <= 0) {
      logs.push({ id: logId(), text: `${actor.name} 的目標已倒下，行動跳過`, type: 'info' })
      continue
    }

    let attackBonus = actor.martial + actor.weaponAtk
    let actionLabel = '普通攻擊'
    let useWisdom = false

    if (action.type === 'skill' && action.skillId) {
      const skill = allItems.find((i) => i.id === action.skillId)
      const cost = parseCost(skill)
      const bonus = parseSkillBonus(skill)
      actionLabel = `武學：${action.skillName ?? skill?.name ?? '?'}`

      // Deduct inner force
      if (!actor.isEnemy) {
        itemRemovals.push({ playerId: actor.id, itemId: 'inner_force_point', quantity: cost })
      }

      attackBonus += bonus

      // Cultivation match bonus
      if (actor.cultivationWeaponMatch) {
        attackBonus += actor.cultivationBonus
      }

      // Hidden weapon uses wisdom
      if (actor.weaponType === 'hidden') {
        attackBonus = actor.wisdom + actor.weaponAtk + bonus
        useWisdom = true
      }
    }

    const attrLabel = useWisdom ? '機智' : '武功'
    const attrValue = useWisdom ? actor.wisdom : actor.martial

    // Defense calculation
    const isTargetDefending = defending.has(target.id)
    const targetBaseDef = Math.floor((target.isEnemy ? target.martial : target.martial) / 2)
    const targetArmorDef = target.armorDef
    const defenseValue = targetBaseDef + targetArmorDef * (isTargetDefending ? 2 : 1)

    // Log the action
    logs.push({
      id: logId(),
      text: `${actor.name} 使用 ${actionLabel} 攻擊 ${target.name}`,
      type: 'action',
    })

    // Dice roll needed
    diceRolls.push({
      purpose: `${actor.name} ${actionLabel} → ${target.name}`,
      formula: '2d6',
    })

    // The actual damage will be calculated when dice results come back
    // For now, store the calculation parameters as a log placeholder
    logs.push({
      id: logId(),
      text: `2d6 + ${attrLabel}${attrValue} + 武器${actor.weaponAtk}${action.type === 'skill' ? ` + 武學加成` : ''} vs 防禦${defenseValue}${isTargetDefending ? '(防禦姿態)' : ''}`,
      type: 'result',
    })
  }

  return { logs, hpChanges, itemRemovals, diceRolls }
}

/**
 * Given a dice total and the attack parameters, compute actual damage.
 */
export function computeDamage(
  diceTotal: number,
  attackBonus: number,
  defenseValue: number,
): { attackValue: number; damage: number } {
  const attackValue = diceTotal + attackBonus
  const damage = Math.max(0, attackValue - defenseValue)
  return { attackValue, damage }
}
