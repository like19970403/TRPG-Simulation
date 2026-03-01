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

export function PlayerTopBar({
  scenarioTitle,
  connectionStatus,
}: PlayerTopBarProps) {
  return (
    <div className="flex h-14 items-center justify-between bg-bg-sidebar px-6">
      <div className="flex items-center gap-3">
        <span className="font-display text-lg font-bold text-gold">TRPG</span>
        <div className="h-5 w-px bg-border" />
        <span className="text-sm text-text-secondary">{scenarioTitle}</span>
      </div>

      <div className="flex items-center gap-2">
        <div
          className={cn('h-2 w-2 rounded-full', statusColors[connectionStatus])}
        />
        <span className="text-xs text-text-tertiary capitalize">
          {connectionStatus}
        </span>
      </div>
    </div>
  )
}
