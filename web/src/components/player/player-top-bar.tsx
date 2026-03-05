import type { ConnectionStatus } from '../../api/types'
import { cn } from '../../lib/cn'

interface PlayerTopBarProps {
  scenarioTitle: string
  connectionStatus: ConnectionStatus
}

const statusColors: Record<ConnectionStatus, string> = {
  connected: 'bg-green-500',
  reconnecting: 'bg-yellow-500',
  connecting: 'bg-yellow-500',
  disconnected: 'bg-gray-500',
}

const statusLabels: Record<ConnectionStatus, string> = {
  connected: '已連線',
  reconnecting: '重新連線中',
  connecting: '連線中',
  disconnected: '已斷線',
}

export function PlayerTopBar({
  scenarioTitle,
  connectionStatus,
}: PlayerTopBarProps) {
  return (
    <div className="flex h-14 items-center justify-between bg-bg-sidebar px-6">
      <div className="flex min-w-0 items-center gap-3">
        <span className="shrink-0 font-display text-lg font-bold text-gold">TRPG</span>
        <div className="h-5 w-px shrink-0 bg-border" />
        <span className="min-w-0 truncate text-sm text-text-secondary">{scenarioTitle}</span>
      </div>

      <div className="flex items-center gap-2">
        <div
          className={cn('h-2 w-2 rounded-full', statusColors[connectionStatus])}
        />
        <span className="text-xs text-text-tertiary">
          {statusLabels[connectionStatus]}
        </span>
      </div>
    </div>
  )
}
