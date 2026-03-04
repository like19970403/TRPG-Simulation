import { useState } from 'react'
import { useParams } from 'react-router'
import { useGameSocket } from '../hooks/use-game-socket'
import { useGameStore } from '../stores/game-store'
import { PlayerTopBar } from '../components/player/player-top-bar'
import { InventorySidebar } from '../components/player/inventory-sidebar'
import { SceneView } from '../components/player/scene-view'
import { GmBroadcastToast } from '../components/player/gm-broadcast-toast'
import { ItemDetailModal } from '../components/player/item-detail-modal'
import { GameStatusOverlay } from '../components/player/game-status-overlay'
import { LoadingSpinner } from '../components/ui/loading-spinner'
import { ConnectionIndicator } from '../components/connection-indicator'
import { cn } from '../lib/cn'
import type { Item } from '../api/types'

export function PlayerGamePage() {
  const { id } = useParams<{ id: string }>()
  const { sendAction, connectionStatus, error } = useGameSocket(id!)
  const [selectedItem, setSelectedItem] = useState<{ item: Item; quantity: number } | null>(null)
  const [sidebarOpen, setSidebarOpen] = useState(false)

  const scenarioContent = useGameStore((s) => s.scenarioContent)
  const gameState = useGameStore((s) => s.gameState)

  // Loading state
  if (!gameState) {
    return (
      <div className="flex h-screen flex-col items-center justify-center gap-4 bg-bg-page">
        {error ? (
          <div className="text-center">
            <p className="text-error">{error}</p>
            <p className="mt-2 text-sm text-text-tertiary">
              無法連線至遊戲場次
            </p>
          </div>
        ) : (
          <>
            <LoadingSpinner className="h-8 w-8 text-gold" />
            <p className="text-sm text-text-tertiary">
              正在連線至遊戲場次...
            </p>
          </>
        )}
      </div>
    )
  }

  const scenarioTitle = scenarioContent?.title ?? '未命名劇本'

  return (
    <div className="flex h-screen flex-col bg-bg-page">
      <ConnectionIndicator status={connectionStatus} />

      {/* Top bar */}
      <PlayerTopBar
        scenarioTitle={scenarioTitle}
        connectionStatus={connectionStatus}
      />

      <div className="h-px bg-border" />

      {/* Mobile sidebar toggle */}
      <button
        className="flex items-center gap-2 border-b border-border bg-bg-sidebar px-4 py-2 text-xs text-text-tertiary md:hidden"
        onClick={() => setSidebarOpen((v) => !v)}
      >
        <span>{sidebarOpen ? '▾' : '▸'}</span>
        <span>背包 / NPC / 筆記</span>
      </button>

      {/* Main area */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left: Inventory sidebar — desktop: always visible, mobile: collapsible */}
        <div
          className={cn(
            'shrink-0 overflow-y-auto border-r border-border md:block',
            sidebarOpen ? 'block' : 'hidden',
          )}
        >
          <InventorySidebar onItemClick={(item, quantity) => setSelectedItem({ item, quantity })} />
        </div>

        {/* Center: Scene view */}
        <div className="flex flex-1 items-start justify-center overflow-y-auto p-4 md:p-8">
          <SceneView sendAction={sendAction} />
        </div>
      </div>

      {/* Overlays & modals */}
      <ItemDetailModal
        item={selectedItem?.item ?? null}
        quantity={selectedItem?.quantity}
        open={!!selectedItem}
        onClose={() => setSelectedItem(null)}
      />
      <GmBroadcastToast />
      <GameStatusOverlay />
    </div>
  )
}
