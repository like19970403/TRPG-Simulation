import { useState } from 'react'
import { useGameStore } from '../../stores/game-store'
import { useAuthStore } from '../../stores/auth-store'
import { NotesPanel } from '../ui/notes-panel'
import { Markdown } from '../ui/markdown'
import { NpcDetailModal } from './npc-detail-modal'
import { CharacterInfoPanel } from './character-info-panel'
import { ITEM_TYPE_LABELS } from '../../lib/scenario-labels'
import type { Item, InventoryEntry, NPC, NPCField, Scene } from '../../api/types'

const EMPTY_INVENTORY: InventoryEntry[] = []
const EMPTY_ITEMS: Item[] = []
const EMPTY_NPCS: NPC[] = []
const EMPTY_NPC_MAP: Record<string, string[]> = {}
const HIDDEN_TYPES = new Set(['martial_skill', 'cultivation_method', 'weapon'])
const HIDDEN_IDS = new Set(['inner_force_point'])

interface InventorySidebarProps {
  onItemClick: (item: Item, quantity: number) => void
}

export function InventorySidebar({ onItemClick }: InventorySidebarProps) {
  const [selectedNpc, setSelectedNpc] = useState<{ npc: NPC; fields: NPCField[] } | null>(null)
  const sessionId = useGameStore((s) => s.session?.id)
  const user = useAuthStore((s) => s.user)
  const inventory = useGameStore(
    (s) =>
      (user ? s.gameState?.player_inventory[user.id] : null) ??
      EMPTY_INVENTORY,
  )
  const allItems = useGameStore(
    (s) => s.scenarioContent?.items ?? EMPTY_ITEMS,
  )
  const allNpcs = useGameStore(
    (s) => s.scenarioContent?.npcs ?? EMPTY_NPCS,
  )
  const revealedNpcFields = useGameStore(
    (s) =>
      (user ? s.gameState?.revealed_npc_fields[user.id] : null) ??
      EMPTY_NPC_MAP,
  )
  const currentScene: Scene | undefined = useGameStore((s) => {
    const sceneId = s.gameState?.current_scene
    return s.scenarioContent?.scenes?.find((sc) => sc.id === sceneId)
  })

  // Resolve inventory entries to item definitions
  const inventoryItems = inventory
    .map((entry) => {
      const item = allItems.find((i) => i.id === entry.item_id)
      return item ? { item, quantity: entry.quantity } : null
    })
    .filter(Boolean) as { item: Item; quantity: number }[]

  // Only show NPCs present in the current scene
  const sceneNpcIds = currentScene?.npcs_present ?? []
  const sceneNpcs = allNpcs.filter((npc) => sceneNpcIds.includes(npc.id))

  // A field is player-visible if it's not hidden/gm_only, OR if it was explicitly revealed by the GM.
  const isFieldVisible = (f: { key: string; visibility: string }, revealedKeys: string[]) =>
    (f.visibility !== 'hidden' && f.visibility !== 'gm_only') || revealedKeys.includes(f.key)

  // Filter to NPCs with at least one visible field
  const visibleNpcs = sceneNpcs.filter((npc) => {
    if (!npc.fields) return false
    const revealedKeys = revealedNpcFields[npc.id] ?? []
    return npc.fields.some((f) => isFieldVisible(f, revealedKeys))
  })

  return (
    <div className="flex w-full min-w-0 flex-col overflow-y-auto bg-bg-sidebar safe-x md:w-60">
      {/* Character info panel */}
      <CharacterInfoPanel />

      {/* Items section — only regular items (skills/cultivation/weapons shown in character card) */}
      <div className="border-b border-border p-4">
        {(() => {
          const bagItems = inventoryItems.filter(({ item }) => !HIDDEN_TYPES.has(item.type) && !HIDDEN_IDS.has(item.id))

          return (
            <>
              <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-text-tertiary">
                背包
              </h3>
              {bagItems.length === 0 && (
                <p className="text-xs text-text-tertiary">背包是空的</p>
              )}
              <div className="flex flex-col gap-1">
                {bagItems.map(({ item, quantity }) => (
                  <button
                    key={item.id}
                    className="rounded-lg px-3 py-2 text-left text-sm text-text-secondary transition-colors hover:bg-bg-card hover:text-text-primary"
                    onClick={() => onItemClick(item, quantity)}
                  >
                    <div className="flex items-center gap-1">
                      <span className="font-medium">{item.name}</span>
                      {quantity > 1 && (
                        <span className="rounded bg-gold/20 px-1 py-0.5 text-xs text-gold">
                          x{quantity}
                        </span>
                      )}
                    </div>
                    <div className="text-xs text-text-tertiary">
                      {ITEM_TYPE_LABELS[item.type] ?? item.type}
                    </div>
                  </button>
                ))}
              </div>
            </>
          )
        })()}
      </div>

      {/* NPCs section */}
      <div className="border-b border-border p-4">
        <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-text-tertiary">
          NPCs
        </h3>
        {visibleNpcs.length === 0 ? (
          <p className="text-xs text-text-tertiary">尚未遇到 NPC</p>
        ) : (
          <div className="flex flex-col gap-3">
            {visibleNpcs.map((npc) => {
              const revealedKeys = revealedNpcFields[npc.id] ?? []
              const visibleFields = (npc.fields ?? []).filter((f) =>
                isFieldVisible(f, revealedKeys),
              )
              return (
                <button
                  key={npc.id}
                  className="w-full min-w-0 cursor-pointer rounded-lg p-2 text-left transition-colors hover:bg-bg-card"
                  onClick={() => setSelectedNpc({ npc, fields: visibleFields })}
                >
                  {npc.image && (
                    <img
                      src={npc.image}
                      alt={npc.name}
                      className="mb-2 h-20 w-20 rounded-lg object-cover"
                    />
                  )}
                  <div className="mb-1 text-sm font-medium text-text-primary">
                    {npc.name}
                  </div>
                  <div className="flex flex-col gap-0.5">
                    {visibleFields.map((f) => (
                      <div key={f.key} className="min-w-0 text-xs text-text-secondary">
                        <span className="text-text-tertiary">{f.label}:</span>{' '}
                        <Markdown className="inline text-xs text-text-secondary [&>p]:inline">{f.value}</Markdown>
                      </div>
                    ))}
                  </div>
                </button>
              )
            })}
          </div>
        )}
      </div>

      {/* Notes section */}
      <div className="flex min-h-40 flex-1 flex-col border-b border-border p-4">
        <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-text-tertiary">
          筆記
        </h3>
        {sessionId ? (
          <NotesPanel
            sessionId={sessionId}
            className="flex flex-1 flex-col"
          />
        ) : (
          <p className="text-xs text-text-tertiary">尚無活動場次</p>
        )}
      </div>

      <NpcDetailModal
        npc={selectedNpc?.npc ?? null}
        revealedFields={selectedNpc?.fields ?? []}
        open={!!selectedNpc}
        onClose={() => setSelectedNpc(null)}
      />
    </div>
  )
}
