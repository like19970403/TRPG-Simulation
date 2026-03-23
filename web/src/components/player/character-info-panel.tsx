import { useState } from 'react'
import { useGameStore } from '../../stores/game-store'
import { useAuthStore } from '../../stores/auth-store'
import { HpBar } from '../combat/hp-bar'
import { CharacterCardModal } from './character-card-modal'
import type { Item } from '../../api/types'

interface CharacterInfoPanelProps {
  sendAction?: (type: string, payload: Record<string, unknown>) => void
}

export function CharacterInfoPanel({ sendAction }: CharacterInfoPanelProps = {}) {
  const [showModal, setShowModal] = useState(false)
  const user = useAuthStore((s) => s.user)
  const gameState = useGameStore((s) => s.gameState)
  const allItems = useGameStore((s) => s.scenarioContent?.items ?? [])

  const userId = user?.id
  if (!userId || !gameState) return null

  const playerState = gameState.players?.[userId]
  if (!playerState?.character_name) return null

  const charName = playerState.character_name || playerState.username
  const inventory = gameState.player_inventory?.[userId] ?? []
  const allWeapons = inventory.map((e) => allItems.find((i: Item) => i.id === e.item_id)).filter((i): i is Item => !!i && i.slot === 'weapon')

  const playerKeys = Object.keys(gameState.players ?? {}).filter((uid) => gameState.players?.[uid]?.character_name)
  const playerIdx = playerKeys.indexOf(userId)
  const equippedId = gameState.variables?.[`equipped_weapon_player${playerIdx + 1}`] as string | undefined
  const weapon = (equippedId ? allWeapons.find((w) => w.id === equippedId) : allWeapons[0]) ?? allWeapons[0]
  const hp = playerIdx >= 0 ? Number(gameState.variables?.[`hp_player${playerIdx + 1}`] ?? 0) : 0
  const attrs = gameState.player_attributes?.[userId] ?? {}
  const maxHp = 10 + Number(attrs['內力'] ?? 5) * 2

  return (
    <>
      <div className="border-b border-border px-4 py-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="font-display text-sm font-bold text-text-primary">{charName}</span>
            {weapon && <span className="text-[9px] text-text-tertiary">{weapon.name}</span>}
          </div>
          <button
            type="button"
            onClick={() => setShowModal(true)}
            className="rounded border border-border px-2 py-1 text-[9px] text-text-tertiary transition-colors hover:border-gold hover:text-gold"
          >
            角色卡
          </button>
        </div>
        {hp > 0 && (
          <div className="mt-1.5">
            <HpBar current={hp} max={maxHp} height="h-2" />
          </div>
        )}
      </div>

      {showModal && (
        <CharacterCardModal userId={userId} onClose={() => setShowModal(false)} sendAction={sendAction} />
      )}
    </>
  )
}
