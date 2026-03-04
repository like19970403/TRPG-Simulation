import type { ConnectionStatus } from '../api/types'

const config: Record<
  ConnectionStatus,
  { dot: string; bg: string; text: string; label: string } | null
> = {
  connected: null,
  connecting: {
    dot: 'bg-yellow-400',
    bg: 'bg-yellow-600/20',
    text: 'text-yellow-400',
    label: '連線中...',
  },
  reconnecting: {
    dot: 'bg-yellow-400 animate-pulse',
    bg: 'bg-yellow-600/20',
    text: 'text-yellow-400',
    label: '正在重新連線...',
  },
  disconnected: {
    dot: 'bg-red-400',
    bg: 'bg-red-600/20',
    text: 'text-red-400',
    label: '已斷線',
  },
}

export function ConnectionIndicator({
  status,
}: {
  status: ConnectionStatus
}) {
  const cfg = config[status]
  if (!cfg) return null

  return (
    <div
      className={`flex items-center justify-center gap-2 px-4 py-1.5 text-xs ${cfg.bg} ${cfg.text}`}
    >
      <span className={`inline-block h-2 w-2 rounded-full ${cfg.dot}`} />
      {cfg.label}
    </div>
  )
}
