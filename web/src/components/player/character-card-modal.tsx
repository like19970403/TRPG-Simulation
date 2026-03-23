import { useGameStore } from '../../stores/game-store'
import { HpBar } from '../combat/hp-bar'
import { parseSkillCost } from '../../lib/combat-utils'
import type { Item } from '../../api/types'
import { cn } from '../../lib/cn'

const ATTR_ORDER = ['武功', '內力', '身法', '機智']

interface CharacterCardModalProps {
  userId: string
  onClose: () => void
  sendAction?: (type: string, payload: Record<string, unknown>) => void
}

export function CharacterCardModal({ userId, onClose, sendAction }: CharacterCardModalProps) {
  const gameState = useGameStore((s) => s.gameState)
  const allItems = useGameStore((s) => s.scenarioContent?.items ?? [])

  if (!gameState) return null

  const playerState = gameState.players?.[userId]
  if (!playerState) return null

  const attrs = gameState.player_attributes?.[userId] ?? {}
  const charName = playerState.character_name || playerState.username
  const inventory = gameState.player_inventory?.[userId] ?? []

  const resolve = (itemId: string) => allItems.find((i: Item) => i.id === itemId)

  // All weapons in inventory
  const allWeapons = inventory.map((e) => resolve(e.item_id)).filter((i): i is Item => !!i && i.slot === 'weapon')

  // Equipped weapon — check variable first, fallback to first weapon
  const playerKeys = Object.keys(gameState.players ?? {}).filter((uid) => gameState.players?.[uid]?.character_name)
  const playerIdx = playerKeys.indexOf(userId)
  const equippedVarName = `equipped_weapon_player${playerIdx + 1}`
  const equippedWeaponId = gameState.variables?.[equippedVarName] as string | undefined
  const weapon = (equippedWeaponId ? allWeapons.find((w) => w.id === equippedWeaponId) : allWeapons[0]) ?? allWeapons[0]

  const handleEquipWeapon = (weaponId: string) => {
    if (!sendAction || playerIdx < 0) return
    sendAction('set_variable', { name: equippedVarName, value: weaponId })
  }
  const skills = inventory.map((e) => resolve(e.item_id)).filter((i): i is Item => !!i && i.type === 'martial_skill')
  const cultivation = inventory.map((e) => resolve(e.item_id)).find((i): i is Item => !!i && i.type === 'cultivation_method')
  const innerForce = inventory.find((e) => e.item_id === 'inner_force_point')?.quantity ?? 0

  const hp = playerIdx >= 0 ? Number(gameState.variables?.[`hp_player${playerIdx + 1}`] ?? 0) : 0
  const maxHp = 10 + Number(attrs['內力'] ?? 5) * 2

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-[#0F0F0FCC]"
      onClick={onClose}
    >
      <div
        className="w-full max-w-md bg-bg-card p-6"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="mb-4 flex items-center justify-between">
          <h2 className="font-display text-xl font-bold text-text-primary">{charName}</h2>
          <button
            type="button"
            onClick={onClose}
            className="text-sm text-text-tertiary hover:text-text-primary"
          >
            ✕
          </button>
        </div>

        {/* Attributes — fixed order */}
        {Object.keys(attrs).length > 0 && (
          <div className="mb-4 flex flex-wrap gap-3">
            {ATTR_ORDER.filter((key) => key in attrs).map((key) => (
              <div key={key} className="rounded border border-border px-3 py-1.5 text-center">
                <div className="text-[10px] text-text-tertiary">{key}</div>
                <div className="font-display text-lg font-bold text-gold">{attrs[key] as number}</div>
              </div>
            ))}
          </div>
        )}

        {/* HP */}
        {hp > 0 && (
          <div className="mb-4">
            <div className="mb-1 flex items-center justify-between">
              <span className="text-xs text-text-tertiary">HP</span>
              <span className="text-xs font-medium text-text-secondary">{hp}/{maxHp}</span>
            </div>
            <HpBar current={hp} max={maxHp} height="h-3" showLabel={false} />
          </div>
        )}

        <div className="mb-4 h-px bg-border" />

        {/* Weapon */}
        {allWeapons.length > 0 && (
          <div className="mb-3">
            <span className="text-[10px] font-semibold uppercase tracking-wider text-text-tertiary">
              武器 {allWeapons.length > 1 && '（點擊切換）'}
            </span>
            <div className="mt-1 flex flex-col gap-1">
              {allWeapons.map((w) => {
                const isEquipped = weapon?.id === w.id
                return (
                  <button
                    key={w.id}
                    type="button"
                    disabled={!sendAction || isEquipped}
                    onClick={() => handleEquipWeapon(w.id)}
                    className={cn(
                      'flex items-center gap-2 border px-3 py-1.5 text-left transition-colors',
                      isEquipped
                        ? 'border-gold bg-gold/10'
                        : 'border-border hover:border-text-tertiary',
                    )}
                  >
                    <span className={cn('text-sm', isEquipped ? 'text-gold' : 'text-text-primary')}>{w.name}</span>
                    <span className="text-[10px] text-text-tertiary">atk +{w.atk ?? 0}</span>
                    {w.two_handed && <span className="text-[9px] text-text-tertiary">雙手</span>}
                    {isEquipped && <span className="ml-auto text-[9px] text-gold">裝備中</span>}
                  </button>
                )
              })}
            </div>
          </div>
        )}

        {/* Skills */}
        {skills.length > 0 && (
          <div className="mb-3">
            <span className="text-[10px] font-semibold uppercase tracking-wider text-text-tertiary">武學</span>
            <div className="mt-1 flex flex-col gap-1">
              {skills.map((skill) => {
                const cost = parseSkillCost(skill)
                return (
                  <div key={skill.id} className="flex items-center gap-2">
                    <span className="rounded bg-emerald-900/40 px-1.5 py-0.5 text-[10px] text-emerald-400">
                      {skill.name}
                    </span>
                    <span className="text-[9px] text-text-tertiary">消耗 {cost} 內力</span>
                  </div>
                )
              })}
            </div>
          </div>
        )}

        {/* Cultivation */}
        {cultivation && (
          <div className="mb-3">
            <span className="text-[10px] font-semibold uppercase tracking-wider text-text-tertiary">心法</span>
            <div className="mt-1">
              <span className="rounded bg-amber-900/40 px-1.5 py-0.5 text-[10px] text-amber-400">
                {cultivation.name}
              </span>
              <p className="mt-1 text-[10px] text-text-tertiary">{cultivation.description}</p>
            </div>
          </div>
        )}

        {/* Inner Force */}
        <div className="mb-3">
          <span className="text-[10px] font-semibold uppercase tracking-wider text-text-tertiary">內力點</span>
          <div className="mt-1 flex items-center gap-1">
            <span className="text-lg text-gold">
              {'●'.repeat(Math.min(innerForce, 10))}
              {'○'.repeat(Math.max(0, Number(attrs['內力'] ?? 5) - innerForce))}
            </span>
            <span className="text-xs text-text-tertiary">({innerForce})</span>
          </div>
        </div>
      </div>
    </div>
  )
}
