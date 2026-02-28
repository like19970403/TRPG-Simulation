import { Link } from 'react-router'
import type { ScenarioResponse } from '../../api/types'
import { ScenarioStatusBadge } from './scenario-status-badge'

interface ScenarioCardProps {
  scenario: ScenarioResponse
}

function formatRelativeTime(dateString: string): string {
  const now = new Date()
  const date = new Date(dateString)
  const diffMs = now.getTime() - date.getTime()
  const diffMin = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMin / 60)
  const diffDays = Math.floor(diffHours / 24)

  if (diffMin < 1) return 'Just now'
  if (diffMin < 60) return `Updated ${diffMin}m ago`
  if (diffHours < 24) return `Updated ${diffHours}h ago`
  if (diffDays < 30) return `Updated ${diffDays}d ago`
  return `Updated ${date.toLocaleDateString()}`
}

export function ScenarioCard({ scenario }: ScenarioCardProps) {
  return (
    <Link
      to={`/scenarios/${scenario.id}`}
      className="flex items-center justify-between rounded-lg bg-bg-card p-5 transition-colors hover:bg-bg-input"
    >
      <div className="flex flex-col gap-1.5">
        <span className="text-base font-semibold text-text-primary">
          {scenario.title}
        </span>
        <span className="text-[13px] text-text-secondary">
          {scenario.description || 'No description'}
        </span>
      </div>
      <div className="flex items-center gap-4">
        <ScenarioStatusBadge status={scenario.status} />
        <span className="whitespace-nowrap text-xs text-text-tertiary">
          {formatRelativeTime(scenario.updatedAt)}
        </span>
      </div>
    </Link>
  )
}
