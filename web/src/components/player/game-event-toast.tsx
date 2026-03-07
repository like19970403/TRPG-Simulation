import { useCallback, useEffect, useRef, useState } from 'react'
import { useGameStore } from '../../stores/game-store'
import { useAuthStore } from '../../stores/auth-store'
import { ItemDetailModal } from './item-detail-modal'
import { NpcDetailModal } from './npc-detail-modal'
import type { Item, NPC, NPCField } from '../../api/types'

interface ToastEntry {
  id: string
  type: 'item_given' | 'item_revealed' | 'item_removed' | 'npc_field_revealed'
  title: string
  name: string
  subtitle?: string
  imageUrl?: string
  // For modal opening
  itemId?: string
  npcId?: string
}

const AUTO_DISMISS_MS = 6000
const MAX_VISIBLE = 3

const EVENT_TYPES = new Set([
  'item_given',
  'item_revealed',
  'item_removed',
  'npc_field_revealed',
])

export function GameEventToast() {
  const [toasts, setToasts] = useState<ToastEntry[]>([])
  const lastIndexRef = useRef(0)

  // Modal state
  const [selectedItem, setSelectedItem] = useState<{ item: Item; quantity: number } | null>(null)
  const [selectedNpc, setSelectedNpc] = useState<{ npc: NPC; fields: NPCField[] } | null>(null)

  useEffect(() => {
    const unsubscribe = useGameStore.subscribe((state) => {
      const { eventLog, scenarioContent } = state
      if (eventLog.length <= lastIndexRef.current) return

      const userId = useAuthStore.getState().user?.id
      if (!userId) return

      const items = scenarioContent?.items ?? []
      const npcs = scenarioContent?.npcs ?? []
      const newToasts: ToastEntry[] = []

      for (let i = lastIndexRef.current; i < eventLog.length; i++) {
        const entry = eventLog[i]
        if (!EVENT_TYPES.has(entry.type)) continue

        const payload = entry.payload as Record<string, unknown>
        const playerIds = payload.player_ids as string[] | undefined
        if (!playerIds?.includes(userId)) continue

        const itemId = payload.item_id as string | undefined
        const npcId = payload.npc_id as string | undefined

        let toast: ToastEntry | null = null

        if (entry.type === 'item_given' || entry.type === 'item_revealed' || entry.type === 'item_removed') {
          const item = items.find((it) => it.id === itemId)
          if (!item) continue
          const qty = (payload.quantity as number) ?? 1
          toast = {
            id: entry.id,
            type: entry.type as ToastEntry['type'],
            title: entry.type === 'item_given' ? '\u7372\u5F97\u9053\u5177' : entry.type === 'item_revealed' ? '\u9053\u5177\u5DF2\u63ED\u9732' : '\u9053\u5177\u5DF2\u79FB\u9664',
            name: item.name,
            subtitle: qty > 1 ? `x${qty}` : undefined,
            imageUrl: item.image,
            itemId: item.id,
          }
        } else if (entry.type === 'npc_field_revealed') {
          const npc = npcs.find((n) => n.id === npcId)
          if (!npc) continue
          const fieldKey = payload.field_key as string
          const field = npc.fields?.find((f) => f.key === fieldKey)
          toast = {
            id: entry.id,
            type: 'npc_field_revealed',
            title: 'NPC \u8CC7\u8A0A\u63ED\u9732',
            name: npc.name,
            subtitle: field?.label,
            imageUrl: npc.image,
            npcId: npc.id,
          }
        }

        if (toast) newToasts.push(toast)
      }

      lastIndexRef.current = eventLog.length

      if (newToasts.length > 0) {
        setToasts((prev) => {
          const merged = [...prev, ...newToasts]
          return merged.length > MAX_VISIBLE ? merged.slice(-MAX_VISIBLE) : merged
        })
      }
    })

    return unsubscribe
  }, [])

  // Auto-dismiss
  useEffect(() => {
    if (toasts.length === 0) return
    const timer = setTimeout(() => {
      setToasts((prev) => prev.slice(1))
    }, AUTO_DISMISS_MS)
    return () => clearTimeout(timer)
  }, [toasts])

  const dismiss = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  const handleClick = useCallback((toast: ToastEntry) => {
    const state = useGameStore.getState()
    const items = state.scenarioContent?.items ?? []
    const npcs = state.scenarioContent?.npcs ?? []
    const userId = useAuthStore.getState().user?.id

    if (toast.itemId) {
      const item = items.find((it) => it.id === toast.itemId)
      if (item) {
        const inv = userId ? state.gameState?.player_inventory[userId] ?? [] : []
        const entry = inv.find((e) => e.item_id === item.id)
        setSelectedItem({ item, quantity: entry?.quantity ?? 1 })
      }
    } else if (toast.npcId) {
      const npc = npcs.find((n) => n.id === toast.npcId)
      if (npc) {
        const revealedKeys = userId
          ? state.gameState?.revealed_npc_fields[userId]?.[toast.npcId] ?? []
          : []
        const visibleFields = (npc.fields ?? []).filter(
          (f) =>
            (f.visibility !== 'hidden' && f.visibility !== 'gm_only') ||
            revealedKeys.includes(f.key),
        )
        setSelectedNpc({ npc, fields: visibleFields })
      }
    }

    dismiss(toast.id)
  }, [dismiss])

  return (
    <>
      {toasts.length > 0 && (
        <div className="fixed bottom-2 right-2 z-40 flex w-[calc(100vw-1rem)] flex-col-reverse gap-2 md:bottom-4 md:right-4 md:w-auto">
          {toasts.map((toast) => (
            <div
              key={toast.id}
              className={`flex max-w-sm cursor-pointer items-center gap-3 rounded-lg border px-4 py-3 shadow-lg transition-colors hover:bg-bg-sidebar ${
                toast.type === 'item_removed'
                  ? 'border-border bg-bg-card'
                  : 'border-gold/30 bg-bg-card'
              }`}
              onClick={() => handleClick(toast)}
            >
              {toast.imageUrl && (
                <img
                  src={toast.imageUrl}
                  alt={toast.name}
                  className="h-8 w-8 shrink-0 rounded object-cover"
                />
              )}
              <div className="min-w-0 flex-1">
                <p className="text-xs font-semibold text-gold">{toast.title}</p>
                <p className="truncate text-sm font-medium text-text-primary">
                  {toast.name}
                  {toast.subtitle && (
                    <span className="ml-1 text-xs text-text-tertiary">{toast.subtitle}</span>
                  )}
                </p>
              </div>
              <button
                className="shrink-0 text-text-tertiary hover:text-text-primary"
                onClick={(e) => {
                  e.stopPropagation()
                  dismiss(toast.id)
                }}
                aria-label="dismiss"
              >
                x
              </button>
            </div>
          ))}
        </div>
      )}

      <ItemDetailModal
        item={selectedItem?.item ?? null}
        quantity={selectedItem?.quantity}
        open={!!selectedItem}
        onClose={() => setSelectedItem(null)}
      />
      <NpcDetailModal
        npc={selectedNpc?.npc ?? null}
        revealedFields={selectedNpc?.fields ?? []}
        open={!!selectedNpc}
        onClose={() => setSelectedNpc(null)}
      />
    </>
  )
}
