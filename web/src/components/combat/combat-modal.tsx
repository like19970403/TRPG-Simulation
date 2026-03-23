import { useState, useMemo, useCallback, useEffect } from 'react'
import { useGameStore } from '../../stores/game-store'
import { useAuthStore } from '../../stores/auth-store'
import { HpBar } from './hp-bar'
import { CombatLog } from './combat-log'
import type { CombatLogEntry } from './combat-log'
import { PlayerActionPicker } from './player-action-picker'
import type { CombatAction } from './player-action-picker'
import { GmCombatControls } from './gm-combat-controls'
import { parseSkillCost, parseSkillBonus } from '../../lib/combat-utils'
import type { Item, InventoryEntry } from '../../api/types'

interface CombatModalProps {
  isGm: boolean
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  sendAction: (type: any, payload: any) => void
}

export function CombatModal({ isGm, sendAction }: CombatModalProps) {
  const user = useAuthStore((s) => s.user)
  const variables = useGameStore((s) => s.gameState?.variables ?? {})
  const players = useGameStore((s) => s.gameState?.players ?? {})
  const playerAttributes = useGameStore((s) => s.gameState?.player_attributes ?? {})
  const playerInventory = useGameStore((s) => s.gameState?.player_inventory ?? {})
  const allItems = useGameStore((s) => s.scenarioContent?.items ?? [])

  const [logs, setLogs] = useState<CombatLogEntry[]>([])
  const [myAction, setMyAction] = useState<CombatAction | null>(null)
  const [confirmed, setConfirmed] = useState(false)
  const [executing, setExecuting] = useState(false)

  // Combat state from variables
  const combatStatus = variables.combat_status as string
  const enemyName = (variables.combat_enemy_name as string) ?? '敵人'
  const enemyHp = Number(variables.combat_enemy_hp ?? 0)
  const enemyMaxHp = Number(variables.combat_enemy_max_hp ?? 1)
  const enemyMartial = Number(variables.combat_enemy_martial ?? 5)
  const enemyDef = Number(variables.combat_enemy_def ?? 0)
  const enemyAgility = Number(variables.combat_enemy_agility ?? 5)
  const enemyWeaponAtk = Number(variables.combat_enemy_weapon_atk ?? 2)
  const combatRound = Number(variables.combat_round ?? 1)

  // Build player list sorted by join order (must be before early return for hooks rules)
  const playerList = useMemo(() => {
    try {
      return Object.entries(players)
        .filter(([, ps]) => ps?.character_name)
        .map(([uid, ps], idx) => ({
          userId: uid,
          name: ps?.character_name || ps?.username || uid.slice(0, 8),
          hpVar: `hp_player${idx + 1}`,
          hp: Number(variables[`hp_player${idx + 1}`] ?? 20),
          maxHp: 10 + Number(playerAttributes[uid]?.['內力'] ?? 5) * 2,
          attrs: playerAttributes[uid] ?? {},
          inventory: (playerInventory[uid] ?? []) as InventoryEntry[],
          ready: variables[`combat_ready_player${idx + 1}`] === true,
          actionJson: variables[`combat_action_player${idx + 1}`] as string | undefined,
        }))
    } catch {
      return []
    }
  }, [players, variables, playerAttributes, playerInventory])

  // Find current user's player data
  const myPlayer = playerList.find((p) => p.userId === user?.id)
  const myIdx = playerList.findIndex((p) => p.userId === user?.id)

  // Reset player action state when GM clears ready flag (new round)
  const myReady = myIdx >= 0 ? variables[`combat_ready_player${myIdx + 1}`] : undefined
  useEffect(() => {
    if (myReady === false || myReady === undefined || myReady === '') {
      setConfirmed(false)
      setMyAction(null)
    }
  }, [myReady])

  // Get equipped weapon type from inventory
  const getWeaponType = useCallback((inv: InventoryEntry[]) => {
    for (const entry of inv) {
      const item = allItems.find((i: Item) => i.id === entry.item_id)
      if (item?.slot === 'weapon' && item.weapon_type) return item.weapon_type
    }
    return undefined
  }, [allItems])

  const innerForceCount = useCallback((inv: InventoryEntry[]) => {
    return inv.find((e) => e.item_id === 'inner_force_point')?.quantity ?? 0
  }, [])

  // Player confirms action
  const handleConfirmAction = useCallback((action: CombatAction) => {
    setMyAction(action)
    setConfirmed(true)
    if (myIdx >= 0) {
      sendAction('set_variable', { name: `combat_action_player${myIdx + 1}`, value: JSON.stringify(action) })
      sendAction('set_variable', { name: `combat_ready_player${myIdx + 1}`, value: true })
    }
  }, [myIdx, sendAction])

  const handleModifyAction = useCallback(() => {
    setConfirmed(false)
    setMyAction(null)
    if (myIdx >= 0) {
      sendAction('set_variable', { name: `combat_ready_player${myIdx + 1}`, value: false })
    }
  }, [myIdx, sendAction])

  // GM: handle enemy action
  const handleEnemyAction = useCallback((action: { type: string; target: string }) => {
    sendAction('set_variable', { name: 'combat_action_enemy', value: JSON.stringify(action) })
    sendAction('set_variable', { name: 'combat_ready_enemy', value: true })
  }, [sendAction])

  // GM: execute round
  const handleExecuteRound = useCallback(async () => {
    setExecuting(true)
    const newLogs: CombatLogEntry[] = [...logs]
    const logId = () => `log-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`

    // Collect all combatants with their actions
    interface ActionEntry {
      name: string
      agility: number
      isEnemy: boolean
      userId?: string
      martial: number
      weaponAtk: number
      weaponType?: string
      armorDef: number
      cultivationBonus: number
      action: CombatAction
      hpVar: string
      currentHp: number
      wisdom: number
    }

    const entries: ActionEntry[] = []

    // Players
    for (let i = 0; i < playerList.length; i++) {
      const p = playerList[i]
      const actionStr = variables[`combat_action_player${i + 1}`] as string | undefined
      let action: CombatAction = { type: 'defend' }
      try { if (actionStr) action = JSON.parse(actionStr) } catch { /* use defend */ }

      const weaponItem = p.inventory
        .map((e) => allItems.find((it: Item) => it.id === e.item_id))
        .find((it): it is Item => !!it && it.slot === 'weapon')

      entries.push({
        name: p.name,
        agility: Number(p.attrs['身法'] ?? 5),
        isEnemy: false,
        userId: p.userId,
        martial: Number(p.attrs['武功'] ?? 5),
        weaponAtk: weaponItem?.atk ?? 0,
        weaponType: weaponItem?.weapon_type,
        armorDef: p.inventory
          .map((e) => allItems.find((it: Item) => it.id === e.item_id))
          .filter((it): it is Item => !!it && !!it.def)
          .reduce((sum, it) => sum + (it.def ?? 0), 0),
        cultivationBonus: 1,
        action,
        hpVar: p.hpVar,
        currentHp: p.hp,
        wisdom: Number(p.attrs['機智'] ?? 5),
      })
    }

    // Enemy
    const enemyActionStr = variables.combat_action_enemy as string | undefined
    let enemyAction: CombatAction = { type: 'attack', target: playerList[0]?.userId }
    try { if (enemyActionStr) enemyAction = JSON.parse(enemyActionStr) } catch { /* default */ }

    entries.push({
      name: enemyName,
      agility: enemyAgility,
      isEnemy: true,
      martial: enemyMartial,
      weaponAtk: enemyWeaponAtk,
      armorDef: enemyDef,
      cultivationBonus: 0,
      action: enemyAction,
      hpVar: 'combat_enemy_hp',
      currentHp: enemyHp,
      wisdom: 5,
    })

    // Sort by agility
    entries.sort((a, b) => b.agility - a.agility)

    // Track HP
    const hpTracker: Record<string, number> = {}
    for (const e of entries) {
      hpTracker[e.hpVar] = e.currentHp
    }
    const defending = new Set(entries.filter((e) => e.action.type === 'defend').map((e) => e.hpVar))

    // Process each action
    for (const actor of entries) {
      if (hpTracker[actor.hpVar] <= 0) continue

      if (actor.action.type === 'defend') {
        newLogs.push({ id: logId(), text: `[回合${combatRound}] ${actor.name} 選擇防禦`, type: 'info' })
        continue
      }

      if (actor.action.type === 'flee') {
        // Roll for flee
        sendAction('dice_roll', { formula: '2d6', purpose: `${actor.name} 嘗試逃跑` })
        newLogs.push({ id: logId(), text: `[回合${combatRound}] ${actor.name} 嘗試逃跑`, type: 'action' })
        await delay(1500)
        continue
      }

      if (actor.action.type === 'item') {
        newLogs.push({ id: logId(), text: `[回合${combatRound}] ${actor.name} 使用道具`, type: 'info' })
        // Deduct consumable item if player
        if (!actor.isEnemy && actor.userId && actor.action.skillId) {
          sendAction('remove_item', { item_id: actor.action.skillId, player_id: actor.userId, quantity: 1 })
        }
        continue
      }

      // Attack or Skill
      const targetEntry = actor.action.target === 'enemy' || actor.isEnemy === false
        ? entries.find((e) => e.isEnemy)
        : entries.find((e) => e.userId === actor.action.target || e.name === actor.action.target)

      if (!targetEntry || hpTracker[targetEntry.hpVar] <= 0) {
        newLogs.push({ id: logId(), text: `[回合${combatRound}] ${actor.name} 的目標已倒下`, type: 'info' })
        continue
      }

      // If skill, deduct inner force
      let skillBonus = 0
      let actionLabel = '普通攻擊'
      if (actor.action.type === 'skill' && actor.action.skillId) {
        const skill = allItems.find((i: Item) => i.id === actor.action.skillId)
        const cost = parseSkillCost(skill)
        actionLabel = actor.action.skillName ?? skill?.name ?? '武學'
        // Only deduct inner force for players, not enemies
        if (!actor.isEnemy && actor.userId) {
          sendAction('remove_item', { item_id: 'inner_force_point', player_id: actor.userId, quantity: cost })
        }
        skillBonus = parseSkillBonus(skill) || 2
      }

      const attackBonus = actor.martial + actor.weaponAtk + skillBonus + (actor.action.type === 'skill' ? actor.cultivationBonus : 0)
      const isTargetDefending = defending.has(targetEntry.hpVar)
      const targetBaseDef = Math.floor(targetEntry.martial / 2)
      const defenseValue = targetBaseDef + targetEntry.armorDef * (isTargetDefending ? 2 : 1)

      // Roll dice
      newLogs.push({ id: logId(), text: `[回合${combatRound}] ${actor.name} 使用 ${actionLabel} 攻擊 ${targetEntry.name}`, type: 'action' })
      sendAction('dice_roll', { formula: '2d6', purpose: `${actor.name} ${actionLabel}` })
      await delay(1000)

      // Get latest dice result
      const diceHistory = useGameStore.getState().gameState?.dice_history ?? []
      const lastDice = diceHistory[diceHistory.length - 1]
      if (!lastDice) {
        newLogs.push({ id: logId(), text: `[回合${combatRound}] ${actor.name} 的骰子結果未就緒，跳過`, type: 'info' })
        continue
      }
      const diceTotal = lastDice.total

      const attackValue = diceTotal + attackBonus
      const damage = Math.max(0, attackValue - defenseValue)
      const oldHp = hpTracker[targetEntry.hpVar]
      const newHp = Math.max(0, oldHp - damage)
      hpTracker[targetEntry.hpVar] = newHp

      newLogs.push({ id: logId(), text: `擲骰 2d6=${diceTotal} + 加成${attackBonus} = 攻擊值 ${attackValue}`, type: 'result' })
      newLogs.push({ id: logId(), text: `${targetEntry.name} 防禦值 = ${defenseValue}${isTargetDefending ? '(防禦姿態)' : ''}`, type: 'result' })
      newLogs.push({ id: logId(), text: `造成 ${damage} 點傷害！${targetEntry.name} HP ${oldHp}→${newHp}`, type: 'damage' })

      // Update HP
      sendAction('set_variable', { name: targetEntry.hpVar, value: newHp })

      // Broadcast result
      sendAction('gm_broadcast', {
        content: `**${actor.name}** 使用 ${actionLabel} 攻擊 ${targetEntry.name}！\n擲骰 2d6=${diceTotal}+${attackBonus}=${attackValue} vs 防禦${defenseValue}\n**造成 ${damage} 點傷害！** HP ${oldHp}→${newHp}`,
      })

      await delay(1500)

      // Check if target is dead
      if (newHp <= 0) {
        newLogs.push({ id: logId(), text: `${targetEntry.name} 倒下了！`, type: 'damage' })
        if (targetEntry.isEnemy) {
          // Auto end combat
          sendAction('gm_broadcast', { content: `**${targetEntry.name} 被擊敗了！戰鬥結束！**` })
        }
      }
    }

    // Reset for next round
    for (let i = 0; i < playerList.length; i++) {
      sendAction('set_variable', { name: `combat_ready_player${i + 1}`, value: false })
      sendAction('set_variable', { name: `combat_action_player${i + 1}`, value: '' })
    }
    sendAction('set_variable', { name: 'combat_ready_enemy', value: false })
    sendAction('set_variable', { name: 'combat_action_enemy', value: '' })
    sendAction('set_variable', { name: 'combat_round', value: combatRound + 1 })

    setLogs(newLogs)
    setExecuting(false)
    setConfirmed(false)
    setMyAction(null)
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [playerList, variables, allItems, enemyName, enemyAgility, enemyMartial, enemyWeaponAtk, enemyDef, enemyHp, combatRound, sendAction])

  // GM: end combat
  const handleEndCombat = useCallback(() => {
    // Refill inner force points to match 內力 attribute
    for (const p of playerList) {
      const maxForce = Number(p.attrs['內力'] ?? 5)
      const current = p.inventory.find((e) => e.item_id === 'inner_force_point')?.quantity ?? 0
      if (current < maxForce) {
        sendAction('give_item', { item_id: 'inner_force_point', player_id: p.userId, quantity: maxForce - current })
      }
    }
    sendAction('set_variable', { name: 'combat_status', value: '' })
    sendAction('gm_broadcast', { content: '**戰鬥結束！**內力點已補回。' })
    setLogs([])
  }, [playerList, sendAction])

  // Early return AFTER all hooks (React hooks rules)
  if (combatStatus !== 'active') return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-[#0F0F0FCC]">
      <div className="flex max-h-[90vh] w-full max-w-[800px] flex-col gap-4 overflow-y-auto bg-bg-card p-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <h2 className="font-display text-xl font-bold text-text-primary">
            ⚔ 戰鬥 — 第 {combatRound} 回合
          </h2>
          {isGm && (
            <button
              type="button"
              onClick={handleEndCombat}
              className="bg-error px-4 py-2 text-xs font-semibold text-white"
            >
              結束戰鬥
            </button>
          )}
        </div>

        <div className="h-px bg-border" />

        {/* Enemy section */}
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-3">
            <span className="font-display text-lg text-text-primary">{enemyName}</span>
            <span className="text-[10px] text-text-tertiary">武功{enemyMartial} 防禦{enemyDef} 身法{enemyAgility}</span>
          </div>
          <HpBar current={enemyHp} max={enemyMaxHp} height="h-3.5" />
        </div>

        <div className="h-px bg-border" />

        {/* GM view: controls + all player status */}
        {isGm ? (
          <GmCombatControls
            players={playerList.map((p, i) => {
              let actionLabel: string | undefined
              try {
                const a = JSON.parse((variables[`combat_action_player${i + 1}`] as string) || '{}')
                actionLabel = a.type === 'skill' ? `武學：${a.skillName}` : a.type === 'attack' ? '普通攻擊' : a.type === 'defend' ? '防禦' : a.type
              } catch { /* skip */ }
              return {
                userId: p.userId,
                name: p.name,
                ready: p.ready,
                actionLabel,
              }
            })}
            totalPlayers={playerList.length}
            enemyName={enemyName}
            onEnemyAction={handleEnemyAction}
            onExecuteRound={handleExecuteRound}
            onEndCombat={handleEndCombat}
            executing={executing}
          />
        ) : (
          /* Player view: my character + action picker */
          myPlayer && (
            <div className="flex flex-col gap-3">
              <div className="flex flex-col gap-2 bg-bg-sidebar p-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium text-text-primary">
                    我的角色：{myPlayer.name}
                  </span>
                  <span className="text-[10px] text-text-tertiary">
                    武器：{getWeaponType(myPlayer.inventory) ?? '空手'} 內力：{innerForceCount(myPlayer.inventory)}
                  </span>
                </div>
                <HpBar current={myPlayer.hp} max={myPlayer.maxHp} height="h-3" />
              </div>

              <PlayerActionPicker
                inventory={myPlayer.inventory}
                allItems={allItems}
                equippedWeaponType={getWeaponType(myPlayer.inventory)}
                innerForceCount={innerForceCount(myPlayer.inventory)}
                onConfirm={handleConfirmAction}
                confirmed={confirmed}
                onModify={handleModifyAction}
                confirmedAction={myAction}
              />

              {/* Teammates */}
              {playerList.filter((p) => p.userId !== user?.id).length > 0 && (
                <div className="flex flex-col gap-1">
                  <span className="text-[10px] font-semibold text-text-tertiary">隊友狀態</span>
                  {playerList
                    .filter((p) => p.userId !== user?.id)
                    .map((p) => (
                      <div key={p.userId} className="flex items-center gap-2">
                        <span className="text-[11px] text-text-primary">{p.name}</span>
                        <div className="w-24">
                          <HpBar current={p.hp} max={p.maxHp} height="h-2" showLabel={false} />
                        </div>
                        <span className="text-[9px] text-text-tertiary">{p.hp}/{p.maxHp}</span>
                        <span className="text-[9px] text-text-tertiary">
                          {p.ready ? '✓ 已決定' : '思考中...'}
                        </span>
                      </div>
                    ))}
                </div>
              )}
            </div>
          )
        )}

        {/* Player HP overview (GM only) */}
        {isGm && (
          <div className="flex flex-col gap-2">
            {playerList.map((p) => (
              <div key={p.userId} className="flex items-center gap-3">
                <span className="w-16 text-xs text-text-primary">{p.name}</span>
                <div className="flex-1"><HpBar current={p.hp} max={p.maxHp} height="h-2.5" /></div>
              </div>
            ))}
          </div>
        )}

        <div className="h-px bg-border" />

        {/* Combat log */}
        <div className="max-h-40 overflow-y-auto">
          <CombatLog entries={logs} />
        </div>
      </div>
    </div>
  )
}

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}
