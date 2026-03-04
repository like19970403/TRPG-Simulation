import { useCallback, useEffect, useRef, useState } from 'react'
import { useGameStore } from '../../stores/game-store'
import { Markdown } from '../ui/markdown'

interface Toast {
  id: string
  content: string
  imageUrl?: string
}

const AUTO_DISMISS_MS = 8000

export function GmBroadcastToast() {
  const [toasts, setToasts] = useState<Toast[]>([])
  const lastProcessedRef = useRef<string | null>(null)

  // Subscribe to store changes outside of render — avoids set-state-in-effect
  useEffect(() => {
    const unsubscribe = useGameStore.subscribe((state) => {
      const { eventLog } = state
      if (eventLog.length === 0) return

      const latest = eventLog[eventLog.length - 1]
      if (latest.type !== 'gm_broadcast') return
      if (latest.id === lastProcessedRef.current) return

      lastProcessedRef.current = latest.id
      const payload = latest.payload as {
        content?: string
        image_url?: string
      }

      const toast: Toast = {
        id: latest.id,
        content: payload.content ?? '',
        imageUrl: payload.image_url,
      }

      setToasts((prev) => [...prev, toast])
    })

    return unsubscribe
  }, [])

  // Auto-dismiss timers
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

  if (toasts.length === 0) return null

  return (
    <div className="fixed left-1/2 top-4 z-50 flex -translate-x-1/2 flex-col gap-2">
      {toasts.map((toast) => (
        <div
          key={toast.id}
          className="flex max-w-md items-start gap-3 rounded-lg border border-gold/30 bg-bg-card px-4 py-3 shadow-lg"
        >
          <div className="flex-1">
            <p className="mb-1 text-xs font-semibold text-gold">GM</p>
            {toast.content && (
              <Markdown className="text-sm text-text-primary">{toast.content}</Markdown>
            )}
            {toast.imageUrl && (
              <img
                src={toast.imageUrl}
                alt="GM 廣播"
                className="mt-2 max-h-32 rounded"
              />
            )}
          </div>
          <button
            className="text-text-tertiary hover:text-text-primary"
            onClick={() => dismiss(toast.id)}
            aria-label="×"
          >
            ×
          </button>
        </div>
      ))}
    </div>
  )
}
