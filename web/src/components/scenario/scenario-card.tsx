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

  if (diffMin < 1) return '剛剛'
  if (diffMin < 60) return `${diffMin} 分鐘前更新`
  if (diffHours < 24) return `${diffHours} 小時前更新`
  if (diffDays < 30) return `${diffDays} 天前更新`
  return `${date.toLocaleDateString('zh-TW')} 更新`
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
          {scenario.description || '沒有描述'}
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
