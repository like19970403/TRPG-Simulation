import { useToastStore } from '../stores/toast-store'
import type { ToastVariant } from '../stores/toast-store'

const variantStyles: Record<ToastVariant, string> = {
  error: 'bg-red-900/90 border-red-700 text-red-100',
  success: 'bg-green-900/90 border-green-700 text-green-100',
  info: 'bg-bg-surface border-border-default text-text-primary',
}

export function ToastContainer() {
  const toasts = useToastStore((s) => s.toasts)
  const removeToast = useToastStore((s) => s.removeToast)

  if (toasts.length === 0) return null

  return (
    <div className="pointer-events-none fixed right-4 top-4 z-50 flex flex-col gap-2">
      {toasts.map((t) => (
        <div
          key={t.id}
          className={`pointer-events-auto flex items-start gap-2 rounded-lg border px-4 py-3 shadow-lg ${variantStyles[t.variant]}`}
        >
          <span className="flex-1 text-sm">{t.message}</span>
          <button
            className="ml-2 shrink-0 text-xs opacity-60 hover:opacity-100"
            onClick={() => removeToast(t.id)}
          >
            &times;
          </button>
        </div>
      ))}
    </div>
  )
}
