import { useState } from 'react'
import { useGameStore } from '../../stores/game-store'
import { Button } from '../ui/button'
import { cn } from '../../lib/cn'

interface ItemsPanelProps {
  sendAction: (type: string, payload: unknown) => void
}

export function ItemsPanel({ sendAction }: ItemsPanelProps) {
  const currentSceneId = useGameStore((s) => s.gameState?.current_scene)
  const scenarioContent = useGameStore((s) => s.scenarioContent)
  const revealedItems = useGameStore((s) => s.gameState?.revealed_items ?? {})
  const revealedNpcFields = useGameStore(
    (s) => s.gameState?.revealed_npc_fields ?? {},
  )
  const players = useGameStore((s) => s.gameState?.players ?? {})

  const [expandedItem, setExpandedItem] = useState<string | null>(null)
  const [expandedNpc, setExpandedNpc] = useState<string | null>(null)

  const scene = scenarioContent?.scenes?.find((s) => s.id === currentSceneId)
  const sceneItems =
    scene?.items_available
      ?.map((id) => scenarioContent?.items?.find((item) => item.id === id))
      .filter(Boolean) ?? []
  const sceneNpcs =
    scene?.npcs_present
      ?.map((id) => scenarioContent?.npcs?.find((npc) => npc.id === id))
      .filter(Boolean) ?? []

  // Check if an item has been revealed to any player
  const isItemRevealed = (itemId: string) =>
    Object.values(revealedItems).some((items) => items.includes(itemId))

  // Check if an NPC field has been revealed to any player
  const isNpcFieldRevealed = (npcId: string, fieldKey: string) =>
    Object.values(revealedNpcFields).some((npcMap) =>
      npcMap[npcId]?.includes(fieldKey),
    )

  return (
    <div className="flex w-[300px] flex-col gap-6 overflow-y-auto bg-bg-sidebar p-5">
      {/* Items section */}
      <div>
        <h2 className="mb-3 font-display text-sm font-semibold uppercase tracking-wider text-gold">
          Items & Clues
        </h2>
        {sceneItems.length === 0 ? (
          <p className="text-xs text-text-tertiary">No items in this scene</p>
        ) : (
          <ul className="flex flex-col gap-1">
            {sceneItems.map((item) => {
              if (!item) return null
              const revealed = isItemRevealed(item.id)
              const expanded = expandedItem === item.id
              return (
                <li key={item.id}>
                  <button
                    className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-sm hover:bg-bg-input"
                    onClick={() =>
                      setExpandedItem(expanded ? null : item.id)
                    }
                  >
                    <span
                      className={cn(
                        'text-xs',
                        revealed ? 'text-success' : 'text-text-tertiary',
                      )}
                    >
                      {revealed ? '✓' : '▸'}
                    </span>
                    <span className="text-text-primary">{item.name}</span>
                    <span className="ml-auto text-xs text-text-tertiary">
                      {item.type}
                    </span>
                  </button>
                  {expanded && (
                    <div className="ml-6 mt-1 space-y-2 rounded-lg bg-bg-input p-3">
                      <p className="text-xs text-text-secondary">
                        {item.description}
                      </p>
                      <Button
                        variant="primary"
                        size="sm"
                        onClick={() =>
                          sendAction('reveal_item', {
                            item_id: item.id,
                            player_ids: Object.keys(players),
                          })
                        }
                      >
                        Reveal to All
                      </Button>
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
          <p className="text-xs text-text-tertiary">No NPCs in this scene</p>
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
                                    player_ids: Object.keys(players),
                                  })
                                }
                              >
                                Reveal
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
    </div>
  )
}
