import { useEffect, useCallback, useRef } from 'react'
import type { NPC, NPCField } from '../../api/types'
import { Markdown } from '../ui/markdown'
import { useFocusTrap } from '../../hooks/use-focus-trap'

interface NpcDetailModalProps {
  npc: NPC | null
  revealedFields: NPCField[]
  open: boolean
  onClose: () => void
}

export function NpcDetailModal({ npc, revealedFields, open, onClose }: NpcDetailModalProps) {
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

  if (!open || !npc) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center overflow-y-auto bg-[#0F0F0FCC] p-4"
      onClick={onClose}
    >
      <div
        ref={dialogRef}
        className="my-auto flex w-full max-w-[480px] flex-col gap-4 rounded-xl bg-bg-card p-6 md:p-8"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
      >
        {npc.image && (
          <img
            src={npc.image}
            alt={npc.name}
            className="max-h-64 w-full rounded-lg object-contain"
          />
        )}

        <h2 className="font-display text-xl font-semibold text-text-primary">
          {npc.name}
        </h2>

        {revealedFields.length > 0 && (
          <div className="flex flex-col gap-2">
            {revealedFields.map((f) => (
              <div key={f.key} className="text-sm text-text-secondary">
                <span className="font-medium text-text-tertiary">{f.label}:</span>{' '}
                <Markdown className="inline-block text-sm text-text-secondary [&>p]:inline">{f.value}</Markdown>
              </div>
            ))}
          </div>
        )}

        <button
          className="mt-2 self-end text-sm text-text-tertiary hover:text-text-primary"
          onClick={onClose}
        >
          Close
        </button>
      </div>
    </div>
  )
}
