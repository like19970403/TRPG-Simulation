import { type ReactNode, useEffect, useCallback } from 'react'
import { Button } from '../ui/button'
import { cn } from '../../lib/cn'

interface ConfirmModalProps {
  open: boolean
  onClose: () => void
  onConfirm: () => void
  title: string
  description: string
  confirmLabel: string
  confirmVariant?: 'primary' | 'secondary' | 'ghost'
  confirmClassName?: string
  loading?: boolean
  icon?: ReactNode
}

export function ConfirmModal({
  open,
  onClose,
  onConfirm,
  title,
  description,
  confirmLabel,
  confirmVariant = 'primary',
  confirmClassName,
  loading = false,
  icon,
}: ConfirmModalProps) {
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

  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-[#0F0F0FCC]"
      onClick={onClose}
    >
      <div
        className="flex w-full max-w-[480px] flex-col items-center gap-5 rounded-xl bg-bg-card p-8"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
      >
        {icon}
        <h2 className="font-display text-[22px] font-semibold text-text-primary">
          {title}
        </h2>
        <p className="text-center text-sm text-text-secondary">{description}</p>
        <div className="flex w-full gap-3">
          <Button
            variant="ghost"
            className="flex-1"
            onClick={onClose}
            disabled={loading}
          >
            Cancel
          </Button>
          <Button
            variant={confirmVariant}
            className={cn('flex-1', confirmClassName)}
            onClick={onConfirm}
            loading={loading}
          >
            {confirmLabel}
          </Button>
        </div>
      </div>
    </div>
  )
}

