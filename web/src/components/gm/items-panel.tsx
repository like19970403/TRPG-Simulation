import { useState } from 'react'
import { useGameStore } from '../../stores/game-store'
import { Button } from '../ui/button'
import { Select } from '../ui/select'
import { Input } from '../ui/input'
import { cn } from '../../lib/cn'
import { ITEM_TYPE_LABELS } from '../../lib/scenario-labels'
import type { InventoryEntry, PlayerState } from '../../api/types'

interface ItemsPanelProps {
  sendAction: (type: string, payload: unknown) => void
}

const EMPTY_PLAYERS: Record<string, PlayerState> = {}
const EMPTY_NPC_FIELDS: Record<string, Record<string, string[]>> = {}
const EMPTY_INVENTORY: Record<string, InventoryEntry[]> = {}

/** Display name for a player: character name first, fallback to username */
function displayName(player: PlayerState): string {
  return player.character_name || player.username
}

export function ItemsPanel({ sendAction }: ItemsPanelProps) {
  const currentSceneId = useGameStore((s) => s.gameState?.current_scene)
  const scenarioContent = useGameStore((s) => s.scenarioContent)
  const players = useGameStore((s) => s.gameState?.players ?? EMPTY_PLAYERS)
  const revealedNpcFields = useGameStore(
    (s) => s.gameState?.revealed_npc_fields ?? EMPTY_NPC_FIELDS,
  )
  const playerInventory = useGameStore(
    (s) => s.gameState?.player_inventory ?? EMPTY_INVENTORY,
  )

  const [expandedItem, setExpandedItem] = useState<string | null>(null)
  const [expandedNpc, setExpandedNpc] = useState<string | null>(null)
  const [expandedPlayer, setExpandedPlayer] = useState<string | null>(null)
  const [giveTarget, setGiveTarget] = useState<string>('__all__')
  const [giveQty, setGiveQty] = useState(1)

  const scene = scenarioContent?.scenes?.find((s) => s.id === currentSceneId)
  const sceneItems =
    scene?.items_available
      ?.map((id) => scenarioContent?.items?.find((item) => item.id === id))
      .filter(Boolean) ?? []
  const sceneNpcs =
    scene?.npcs_present
      ?.map((id) => scenarioContent?.npcs?.find((npc) => npc.id === id))
      .filter(Boolean) ?? []

  const playerIds = Object.keys(players)

  // Check if an NPC field has been revealed to any player
  const isNpcFieldRevealed = (npcId: string, fieldKey: string) =>
    Object.values(revealedNpcFields).some((npcMap) =>
      npcMap[npcId]?.includes(fieldKey),
    )

  const handleGiveItem = (itemId: string, stackable: boolean) => {
    const qty = stackable ? giveQty : 1
    if (giveTarget === '__all__') {
      sendAction('give_item', {
        item_id: itemId,
        player_ids: playerIds,
        quantity: qty,
      })
    } else {
      sendAction('give_item', {
        item_id: itemId,
        player_id: giveTarget,
        quantity: qty,
      })
    }
  }

  const handleRemoveItem = (playerId: string, itemId: string, qty: number) => {
    sendAction('remove_item', {
      item_id: itemId,
      player_id: playerId,
      quantity: qty,
    })
  }

  return (
    <div className="flex w-[300px] flex-col gap-6 overflow-y-auto bg-bg-sidebar p-5">
      {/* Scene items section */}
      <div>
        <h2 className="mb-3 font-display text-sm font-semibold uppercase tracking-wider text-gold">
          場景道具
        </h2>
        {sceneItems.length === 0 ? (
          <p className="text-xs text-text-tertiary">此場景無道具</p>
        ) : (
          <ul className="flex flex-col gap-1">
            {sceneItems.map((item) => {
              if (!item) return null
              const expanded = expandedItem === item.id
              return (
                <li key={item.id}>
                  <button
                    className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-sm hover:bg-bg-input"
                    onClick={() =>
                      setExpandedItem(expanded ? null : item.id)
                    }
                  >
                    <span className="text-xs text-text-tertiary">
                      {expanded ? '▾' : '▸'}
                    </span>
                    <span className="text-text-primary">{item.name}</span>
                    <span className="ml-auto text-xs text-text-tertiary">
                      {ITEM_TYPE_LABELS[item.type] ?? item.type}
                    </span>
                  </button>
                  {expanded && (
                    <div className="ml-6 mt-1 space-y-2 rounded-lg bg-bg-input p-3">
                      <p className="text-xs text-text-secondary">
                        {item.description}
                      </p>
                      {item.gm_notes && (
                        <p className="text-xs text-gold">
                          GM：{item.gm_notes}
                        </p>
                      )}
                      {item.image && (
                        <img
                          src={item.image}
                          alt={item.name}
                          className="max-h-24 rounded"
                        />
                      )}
                      {/* Give controls */}
                      <div className="flex flex-col gap-1.5">
                        <Select
                          value={giveTarget}
                          onChange={(e) => setGiveTarget(e.target.value)}
                          className="py-1.5 text-xs"
                        >
                          <option value="__all__">全體玩家</option>
                          {playerIds.map((pid) => (
                            <option key={pid} value={pid}>
                              {displayName(players[pid])}
                            </option>
                          ))}
                        </Select>
                        {item.stackable && (
                          <div className="flex items-center gap-1">
                            <span className="text-xs text-text-tertiary">
                              數量
                            </span>
                            <Input
                              type="number"
                              min={1}
                              value={giveQty}
                              onChange={(e) =>
                                setGiveQty(
                                  Math.max(1, parseInt(e.target.value) || 1),
                                )
                              }
                              className="w-16 py-1.5 text-xs"
                            />
                          </div>
                        )}
                        <Button
                          variant="primary"
                          size="sm"
                          onClick={() =>
                            handleGiveItem(item.id, !!item.stackable)
                          }
                        >
                          給予
                        </Button>
                      </div>
                    </div>
                  )}
                </li>
              )
            })}
          </ul>
        )}
      </div>

      {/* NPCs section */}
      <div>
        <h2 className="mb-3 font-display text-sm font-semibold uppercase tracking-wider text-gold">
          NPCs
        </h2>
        {sceneNpcs.length === 0 ? (
          <p className="text-xs text-text-tertiary">此場景無 NPC</p>
        ) : (
          <ul className="flex flex-col gap-1">
            {sceneNpcs.map((npc) => {
              if (!npc) return null
              const expanded = expandedNpc === npc.id
              return (
                <li key={npc.id}>
                  <button
                    className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-sm hover:bg-bg-input"
                    onClick={() =>
                      setExpandedNpc(expanded ? null : npc.id)
                    }
                  >
                    <span className="text-text-tertiary">▸</span>
                    <span className="text-text-primary">{npc.name}</span>
                  </button>
                  {expanded && npc.fields && (
                    <div className="ml-6 mt-1 space-y-1 rounded-lg bg-bg-input p-3">
                      {npc.fields.map((field) => {
                        const revealed = isNpcFieldRevealed(
                          npc.id,
                          field.key,
                        )
                        return (
                          <div
                            key={field.key}
                            className="flex items-center justify-between text-xs"
                          >
                            <span className="text-text-secondary">
                              {field.label}:{' '}
                              {field.visibility === 'hidden' && !revealed
                                ? '???'
                                : field.value}
                            </span>
                            {field.visibility === 'hidden' && !revealed && (
                              <Button
                                variant="ghost"
                                size="sm"
                                className="h-6 px-2 text-xs"
                                onClick={() =>
                                  sendAction('reveal_npc_field', {
                                    npc_id: npc.id,
                                    field_key: field.key,
                                    player_ids: playerIds,
                                  })
                                }
                              >
                                揭露
                              </Button>
                            )}
                          </div>
                        )
                      })}
                    </div>
                  )}
                </li>
              )
            })}
          </ul>
        )}
      </div>

      {/* Player inventory overview */}
      <div>
        <h2 className="mb-3 font-display text-sm font-semibold uppercase tracking-wider text-gold">
          角色背包
        </h2>
        {playerIds.length === 0 ? (
          <p className="text-xs text-text-tertiary">尚無玩家連線</p>
        ) : (
          <ul className="flex flex-col gap-1">
            {playerIds.map((pid) => {
              const player = players[pid]
              const inv = playerInventory[pid] ?? []
              const expanded = expandedPlayer === pid
              return (
                <li key={pid}>
                  <button
                    className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-sm hover:bg-bg-input"
                    onClick={() =>
                      setExpandedPlayer(expanded ? null : pid)
                    }
                  >
                    <span className="text-xs text-text-tertiary">
                      {expanded ? '▾' : '▸'}
                    </span>
                    <span
                      className={cn(
                        'text-text-primary',
                        !player.online && 'opacity-50',
                      )}
                    >
                      {displayName(player)}
                    </span>
                    <span className="ml-auto text-xs text-text-tertiary">
                      {inv.length} 件
                    </span>
                  </button>
                  {expanded && (
                    <div className="ml-6 mt-1 space-y-1 rounded-lg bg-bg-input p-3">
                      {inv.length === 0 ? (
                        <p className="text-xs text-text-tertiary">
                          背包是空的
                        </p>
                      ) : (
                        inv.map((entry) => {
                          const itemDef = scenarioContent?.items?.find(
                            (i) => i.id === entry.item_id,
                          )
                          return (
                            <div
                              key={entry.item_id}
                              className="flex items-center justify-between text-xs"
                            >
                              <span className="text-text-secondary">
                                {itemDef?.name ?? entry.item_id}
                                {entry.quantity > 1 && (
                                  <span className="ml-1 rounded bg-gold/20 px-1 py-0.5 text-gold">
                                    x{entry.quantity}
                                  </span>
                                )}
                              </span>
                              <Button
                                variant="ghost"
                                size="sm"
                                className="h-6 px-2 text-xs text-error hover:text-error"
                                onClick={() =>
                                  handleRemoveItem(pid, entry.item_id, 1)
                                }
                              >
                                移除
                              </Button>
                            </div>
                          )
                        })
                      )}
                    </div>
                  )}
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </div>
  )
}
