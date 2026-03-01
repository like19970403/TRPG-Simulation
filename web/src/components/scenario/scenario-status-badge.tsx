import { cn } from '../../lib/cn'
import type { ScenarioStatus } from '../../api/types'

const statusConfig: Record<ScenarioStatus, { label: string; className: string }> = {
  draft: {
    label: '草稿',
    className: 'bg-text-tertiary/20 text-text-secondary',
  },
  published: {
    label: '已發布',
    className: 'bg-[#4ADE8020] text-success',
  },
  archived: {
    label: '已封存',
    className: 'bg-[#F59E0B20] text-[#F59E0B]',
  },
}

interface ScenarioStatusBadgeProps {
  status: ScenarioStatus
  className?: string
}

export function ScenarioStatusBadge({ status, className }: ScenarioStatusBadgeProps) {
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
