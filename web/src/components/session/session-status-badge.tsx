import { cn } from '../../lib/cn'
import type { SessionStatus } from '../../api/types'

const statusConfig: Record<SessionStatus, { label: string; className: string }> = {
  lobby: {
    label: 'Lobby',
    className: 'bg-[#3B82F620] text-[#60A5FA]',
  },
  active: {
    label: 'Active',
    className: 'bg-[#4ADE8020] text-success',
  },
  paused: {
    label: 'Paused',
    className: 'bg-[#F59E0B20] text-[#F59E0B]',
  },
  completed: {
    label: 'Completed',
    className: 'bg-text-tertiary/20 text-text-secondary',
  },
}

interface SessionStatusBadgeProps {
  status: SessionStatus
  className?: string
}

export function SessionStatusBadge({ status, className }: SessionStatusBadgeProps) {
  const config = statusConfig[status]
  return (
    <span
      className={cn(
        'inline-block rounded-full px-3 py-1 text-xs font-medium',
        config.className,
        className,
      )}
    >
      {config.label}
    </span>
  )
}
