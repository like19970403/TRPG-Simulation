import { useGameStore } from '../../stores/game-store'
import { useAuthStore } from '../../stores/auth-store'
import { NotesPanel } from '../ui/notes-panel'
import type { Item, NPC, Scene } from '../../api/types'

const EMPTY_IDS: string[] = []
const EMPTY_ITEMS: Item[] = []
const EMPTY_NPCS: NPC[] = []
const EMPTY_NPC_MAP: Record<string, string[]> = {}

interface InventorySidebarProps {
  onItemClick: (item: Item) => void
}

export function InventorySidebar({ onItemClick }: InventorySidebarProps) {
  const sessionId = useGameStore((s) => s.session?.id)
  const user = useAuthStore((s) => s.user)
  const revealedItemIds = useGameStore(
    (s) =>
      (user ? s.gameState?.revealed_items[user.id] : null) ?? EMPTY_IDS,
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

  const revealedItems = allItems.filter((item) =>
    revealedItemIds.includes(item.id),
  )

  // Only show NPCs present in the current scene
  const sceneNpcIds = currentScene?.npcs_present ?? []
  const sceneNpcs = allNpcs.filter((npc) => sceneNpcIds.includes(npc.id))

  // Filter to NPCs with at least one visible field (public or revealed hidden)
  const visibleNpcs = sceneNpcs.filter((npc) => {
    if (!npc.fields) return false
    const revealedKeys = revealedNpcFields[npc.id] ?? []
    return npc.fields.some(
      (f) => f.visibility === 'public' || revealedKeys.includes(f.key),
    )
  })

  return (
    <div className="flex w-[240px] flex-col overflow-y-auto bg-bg-sidebar">
      {/* Items section */}
      <div className="border-b border-border p-4">
        <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-text-tertiary">
          道具欄
        </h3>
        {revealedItems.length === 0 ? (
          <p className="text-xs text-text-tertiary">尚無已揭露的道具</p>
        ) : (
          <div className="flex flex-col gap-1">
            {revealedItems.map((item) => (
              <button
                key={item.id}
                className="rounded-lg px-3 py-2 text-left text-sm text-text-secondary transition-colors hover:bg-bg-card hover:text-text-primary"
                onClick={() => onItemClick(item)}
              >
                <div className="font-medium">{item.name}</div>
                <div className="text-xs text-text-tertiary">{item.type}</div>
              </button>
            ))}
          </div>
        )}
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
              const visibleFields = (npc.fields ?? []).filter(
                (f) =>
                  f.visibility === 'public' || revealedKeys.includes(f.key),
              )
              return (
                <div key={npc.id}>
                  <div className="mb-1 text-sm font-medium text-text-primary">
                    {npc.name}
                  </div>
                  <div className="flex flex-col gap-0.5">
                    {visibleFields.map((f) => (
                      <div key={f.key} className="text-xs text-text-secondary">
                        <span className="text-text-tertiary">{f.label}:</span>{' '}
                        {f.value}
                      </div>
                    ))}
                  </div>
                </div>
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
    </div>
  )
}
