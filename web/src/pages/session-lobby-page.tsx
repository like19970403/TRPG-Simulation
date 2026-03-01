import { useState, useEffect, useCallback } from 'react'
import { useParams, useNavigate, Link } from 'react-router'
import { getSession, startSession } from '../api/sessions'
import { getScenario } from '../api/scenarios'
import { useAuthStore } from '../stores/auth-store'
import { SessionStatusBadge } from '../components/session/session-status-badge'
import { SessionPlayerList } from '../components/session/session-player-list'
import { LoadingSpinner } from '../components/ui/loading-spinner'
import { ApiClientError } from '../api/client'
import { ROUTES } from '../lib/constants'
import type { SessionResponse } from '../api/types'

const STATUS_POLL_MS = 3000

export function SessionLobbyPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)

  const [session, setSession] = useState<SessionResponse | null>(null)
  const [scenarioTitle, setScenarioTitle] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [startLoading, setStartLoading] = useState(false)

  const sessionStatus = session?.status
  const sessionGmId = session?.gmId

  // Initial fetch
  useEffect(() => {
    if (!id) return
    setLoading(true)

    getSession(id)
      .then((s) => {
        setSession(s)
        return getScenario(s.scenarioId).then((sc) =>
          setScenarioTitle(sc.title),
        )
      })
      .catch((err) => {
        setError(
          err instanceof ApiClientError
            ? err.body.message
            : 'Failed to load session',
        )
      })
      .finally(() => setLoading(false))
  }, [id])

  // Poll session status for lobby → active transitions
  useEffect(() => {
    if (!id || sessionStatus !== 'lobby') return

    const timer = setInterval(async () => {
      try {
        const updated = await getSession(id)
        setSession(updated)
      } catch {
        // Silently retry on next poll
      }
    }, STATUS_POLL_MS)

    return () => clearInterval(timer)
  }, [id, sessionStatus])

  // Auto-navigate when session becomes active
  useEffect(() => {
    if (!user || !id || sessionStatus !== 'active') return

    if (sessionGmId === user.id) {
      navigate(`/sessions/${id}/gm`, { replace: true })
    } else {
      navigate(`/sessions/${id}/play`, { replace: true })
    }
  }, [sessionStatus, sessionGmId, user, id, navigate])

  const handleStartGame = useCallback(async () => {
    if (!id) return
    setStartLoading(true)
    try {
      const updated = await startSession(id)
      setSession(updated)
    } catch (err) {
      setError(
        err instanceof ApiClientError
          ? err.body.message
          : 'Failed to start game',
      )
    } finally {
      setStartLoading(false)
    }
  }, [id])

  if (loading) {
    return (
      <div className="flex justify-center py-24">
        <LoadingSpinner className="h-8 w-8 text-gold" />
      </div>
    )
  }

  if (error || !session) {
    return (
      <div className="flex flex-col items-center gap-4 py-24">
        <p className="text-sm text-error">{error || 'Session not found'}</p>
        <Link
          to={ROUTES.SESSIONS}
          className="text-sm text-gold hover:text-gold-light"
        >
          Back to Sessions
        </Link>
      </div>
    )
  }

  const isGm = user?.id === session.gmId

  return (
    <div className="flex flex-col gap-8 px-[60px] py-10">
      {/* Back link */}
      <Link
        to={ROUTES.SESSIONS}
        className="text-sm text-text-secondary hover:text-text-primary"
      >
        &larr; Back to Sessions
      </Link>

      {/* Title + Status */}
      <div className="flex items-center justify-between">
        <div className="flex flex-col gap-2">
          <h1 className="font-display text-[28px] font-semibold text-text-primary">
            {scenarioTitle || 'Game Lobby'}
          </h1>
          <div className="flex items-center gap-3">
            <SessionStatusBadge status={session.status} />
            <span className="text-sm text-text-tertiary">
              {isGm ? 'You are the GM' : 'You are a Player'}
            </span>
          </div>
        </div>

        {/* GM: Start Game button (only in lobby) */}
        {isGm && session.status === 'lobby' && (
          <button
            className="rounded-lg bg-gold px-6 py-2.5 text-sm font-medium text-bg-page transition-colors hover:bg-gold/80 disabled:opacity-50"
            onClick={handleStartGame}
            disabled={startLoading}
          >
            {startLoading ? 'Starting...' : 'Start Game'}
          </button>
        )}
      </div>

      {/* Invite Code */}
      <div className="flex flex-col gap-2">
        <h2 className="text-sm font-semibold text-text-secondary">
          Invite Code
        </h2>
        <div className="flex items-center gap-3">
          <span className="font-mono text-2xl tracking-widest text-gold">
            {session.inviteCode}
          </span>
          <button
            className="rounded border border-border px-3 py-1 text-xs text-text-secondary transition-colors hover:text-text-primary"
            onClick={() =>
              navigator.clipboard.writeText(session.inviteCode)
            }
          >
            Copy
          </button>
        </div>
        <p className="text-xs text-text-tertiary">
          Share this code with players so they can join the session.
        </p>
      </div>

      {/* Players */}
      <div className="flex flex-col gap-3">
        <h2 className="text-sm font-semibold text-text-secondary">
          Players
        </h2>
        <SessionPlayerList sessionId={session.id} />
      </div>

      {/* Player waiting message */}
      {!isGm && session.status === 'lobby' && (
        <p className="text-center text-sm text-text-tertiary">
          Waiting for GM to start the game...
        </p>
      )}
    </div>
  )
}
