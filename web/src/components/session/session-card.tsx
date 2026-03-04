import { Link, useNavigate } from 'react-router'
import { SessionStatusBadge } from './session-status-badge'
import type { SessionResponse, SessionStatus } from '../../api/types'

interface SessionCardProps {
  session: SessionResponse
  scenarioTitle?: string
}

function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleDateString('zh-TW', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

function getLobbyPath(session: SessionResponse): string {
  const status = session.status as SessionStatus
  if (status === 'active' || status === 'paused') {
    // Active sessions go directly to game
    return `/sessions/${session.id}/lobby`
  }
  return `/sessions/${session.id}/lobby`
}

export function SessionCard({ session, scenarioTitle }: SessionCardProps) {
  const navigate = useNavigate()

  return (
    <Link
      to={getLobbyPath(session)}
      className="flex flex-col gap-3 rounded-lg border border-border bg-bg-card p-5 transition-colors hover:border-gold/40"
    >
      <div className="flex items-center justify-between">
        <h3 className="font-medium text-text-primary">
          {scenarioTitle ?? '未命名劇本'}
        </h3>
        <SessionStatusBadge status={session.status} />
      </div>

      <div className="flex items-center gap-4 text-xs text-text-tertiary">
        <span>邀請碼：<span className="font-mono text-text-secondary">{session.inviteCode}</span></span>
        <span>建立於 {formatDate(session.createdAt)}</span>
        {session.status === 'completed' && (
          <button
            type="button"
            className="ml-auto text-gold hover:underline"
            onClick={(e) => {
              e.preventDefault()
              e.stopPropagation()
              navigate(`/sessions/${session.id}/replay`)
            }}
          >
            回放
          </button>
        )}
      </div>
    </Link>
  )
}
