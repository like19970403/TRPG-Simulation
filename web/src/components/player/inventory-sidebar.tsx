import { useGameStore } from '../../stores/game-store'
import { useAuthStore } from '../../stores/auth-store'
import type { Item } from '../../api/types'

const EMPTY_IDS: string[] = []
const EMPTY_ITEMS: Item[] = []

interface InventorySidebarProps {
  onItemClick: (item: Item) => void
}

export function InventorySidebar({ onItemClick }: InventorySidebarProps) {
  const user = useAuthStore((s) => s.user)
  const revealedItemIds = useGameStore(
    (s) =>
      (user ? s.gameState?.revealed_items[user.id] : null) ?? EMPTY_IDS,
  )
  const allItems = useGameStore(
    (s) => s.scenarioContent?.items ?? EMPTY_ITEMS,
  )

  const revealedItems = allItems.filter((item) =>
    revealedItemIds.includes(item.id),
  )

  return (
    <div className="flex w-[240px] flex-col bg-bg-sidebar">
      {/* Items section */}
      <div className="border-b border-border p-4">
        <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-text-tertiary">
          Inventory
        </h3>
        {revealedItems.length === 0 ? (
          <p className="text-xs text-text-tertiary">No items revealed yet</p>
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

      {/* Character section — placeholder */}
      <div className="p-4">
        <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-text-tertiary">
          Character
        </h3>
        <p className="text-xs text-text-tertiary">No character assigned</p>
      </div>
    </div>
  )
}
