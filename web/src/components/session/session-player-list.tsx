import { useEffect, useState } from 'react'
import { listSessionPlayers } from '../../api/sessions'
import type { SessionPlayerResponse } from '../../api/types'

interface SessionPlayerListProps {
  sessionId: string
}

const POLL_INTERVAL_MS = 3000

function formatJoinedAt(dateString: string): string {
  return new Date(dateString).toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function SessionPlayerList({ sessionId }: SessionPlayerListProps) {
  const [players, setPlayers] = useState<SessionPlayerResponse[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false

    async function fetchPlayers() {
      try {
        const res = await listSessionPlayers(sessionId)
        if (!cancelled) {
          setPlayers(res.players)
          setLoading(false)
        }
      } catch {
        // Silently retry on next poll
        if (!cancelled) setLoading(false)
      }
    }

    fetchPlayers()
    const timer = setInterval(fetchPlayers, POLL_INTERVAL_MS)

    return () => {
      cancelled = true
      clearInterval(timer)
    }
  }, [sessionId])

  if (loading) {
    return (
      <p className="text-sm text-text-tertiary">Loading players...</p>
    )
  }

  if (players.length === 0) {
    return (
      <p className="text-sm text-text-tertiary">
        No players have joined yet.
      </p>
    )
  }

  return (
    <ul className="flex flex-col gap-2">
      {players.map((player) => (
        <li
          key={player.id}
          className="flex items-center justify-between rounded-md border border-border bg-bg-card px-4 py-2.5"
        >
          <span className="text-sm text-text-primary">
            Player {player.userId.slice(0, 8)}
          </span>
          <span className="text-xs text-text-tertiary">
            Joined {formatJoinedAt(player.joinedAt)}
          </span>
        </li>
      ))}
    </ul>
  )
}
