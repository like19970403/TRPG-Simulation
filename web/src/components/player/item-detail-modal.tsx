import { useEffect, useCallback, useRef } from 'react'
import type { Item } from '../../api/types'
import { Markdown } from '../ui/markdown'
import { useFocusTrap } from '../../hooks/use-focus-trap'

interface ItemDetailModalProps {
  item: Item | null
  quantity?: number
  open: boolean
  onClose: () => void
}

export function ItemDetailModal({ item, quantity, open, onClose }: ItemDetailModalProps) {
  const dialogRef = useRef<HTMLDivElement>(null)
  useFocusTrap(dialogRef, open)

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    },
    [onClose],
  )

  useEffect(() => {
    if (open) {
      document.addEventListener('keydown', handleKeyDown)
      return () => document.removeEventListener('keydown', handleKeyDown)
    }
  }, [open, handleKeyDown])

  if (!open || !item) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-[#0F0F0FCC]"
      onClick={onClose}
    >
      <div
        ref={dialogRef}
        className="flex w-full max-w-[480px] flex-col gap-4 rounded-xl bg-bg-card p-8"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
      >
        {item.image && (
          <img
            src={item.image}
            alt={item.name}
            className="h-48 w-full rounded-lg object-cover"
          />
        )}

        <div className="flex items-center gap-2">
          <h2 className="font-display text-xl font-semibold text-text-primary">
            {item.name}
          </h2>
          <span className="rounded-full bg-gold/20 px-2 py-0.5 text-xs font-medium text-gold">
            {item.type}
          </span>
          {quantity != null && quantity > 1 && (
            <span className="rounded-full bg-gold/20 px-2 py-0.5 text-xs font-medium text-gold">
              x{quantity}
            </span>
          )}
        </div>

        <Markdown className="text-sm leading-relaxed text-text-secondary">
          {item.description}
        </Markdown>

        <button
          className="mt-2 self-end text-sm text-text-tertiary hover:text-text-primary"
          onClick={onClose}
        >
          關閉
        </button>
      </div>
    </div>
  )
}
