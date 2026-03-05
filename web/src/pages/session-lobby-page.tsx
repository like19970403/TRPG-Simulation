import { useState, useEffect, useCallback } from 'react'
import { useParams, useNavigate, Link } from 'react-router'
import { getSession, startSession, deleteSession } from '../api/sessions'
import { getScenario } from '../api/scenarios'
import { listCharacters, assignCharacter } from '../api/characters'
import { useAuthStore } from '../stores/auth-store'
import { SessionStatusBadge } from '../components/session/session-status-badge'
import { SessionPlayerList } from '../components/session/session-player-list'
import { LoadingSpinner } from '../components/ui/loading-spinner'
import { Button } from '../components/ui/button'
import { ApiClientError } from '../api/client'
import { ROUTES } from '../lib/constants'
import { useToastStore } from '../stores/toast-store'
import type { SessionResponse, CharacterResponse } from '../api/types'

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

  // Character selection (Player only)
  const [characters, setCharacters] = useState<CharacterResponse[]>([])
  const [selectedCharId, setSelectedCharId] = useState('')
  const [assignLoading, setAssignLoading] = useState(false)
  const [assignedName, setAssignedName] = useState('')

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
            : '場次載入失敗',
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
          : '開始遊戲失敗',
      )
    } finally {
      setStartLoading(false)
    }
  }, [id])

  // Fetch player's characters for assignment (Player only)
  useEffect(() => {
    if (!session || session.gmId === user?.id) return
    listCharacters(50, 0)
      .then((res) => setCharacters(res.characters))
      .catch(() => {
        // Non-critical: player can still participate without character
      })
  }, [session, user?.id])

  const handleAssignCharacter = useCallback(async () => {
    if (!id || !selectedCharId) return
    setAssignLoading(true)
    try {
      await assignCharacter(id, { characterId: selectedCharId })
      const char = characters.find((c) => c.id === selectedCharId)
      setAssignedName(char?.name ?? '角色')
    } catch (err) {
      setError(
        err instanceof ApiClientError
          ? err.body.message
          : '分配角色失敗',
      )
    } finally {
      setAssignLoading(false)
    }
  }, [id, selectedCharId, characters])

  const handleDeleteSession = useCallback(async () => {
    if (!id || !confirm('確定要刪除此場次？此操作無法復原。')) return
    try {
      await deleteSession(id)
      navigate(ROUTES.SESSIONS, { replace: true })
    } catch (err) {
      setError(
        err instanceof ApiClientError
          ? err.body.message
          : '刪除場次失敗',
      )
    }
  }, [id, navigate])

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
        <p className="text-sm text-error">{error || '找不到場次'}</p>
        <Link
          to={ROUTES.SESSIONS}
          className="text-sm text-gold hover:text-gold-light"
        >
          回到場次列表
        </Link>
      </div>
    )
  }

  const isGm = user?.id === session.gmId

  return (
    <div className="flex flex-col gap-8 px-15 py-10">
      {/* Back link */}
      <Link
        to={ROUTES.SESSIONS}
        className="text-sm text-text-secondary hover:text-text-primary"
      >
        &larr; 回到場次列表
      </Link>

      {/* Title + Status */}
      <div className="flex items-center justify-between">
        <div className="flex flex-col gap-2">
          <h1 className="font-display text-[28px] font-semibold text-text-primary">
            {scenarioTitle || '遊戲大廳'}
          </h1>
          <div className="flex items-center gap-3">
            <SessionStatusBadge status={session.status} />
            <span className="text-sm text-text-tertiary">
              {isGm ? '你是 GM' : '你是玩家'}
            </span>
          </div>
        </div>

        {/* GM: Actions (only in lobby) */}
        {isGm && session.status === 'lobby' && (
          <div className="flex gap-3">
            <button
              className="rounded-lg border border-error px-4 py-2.5 text-sm font-medium text-error transition-colors hover:bg-error/10 cursor-pointer"
              onClick={handleDeleteSession}
            >
              刪除場次
            </button>
            <button
              className="rounded-lg bg-gold px-6 py-2.5 text-sm font-medium text-bg-page transition-colors hover:bg-gold/80 disabled:opacity-50 cursor-pointer"
              onClick={handleStartGame}
              disabled={startLoading}
            >
              {startLoading ? '啟動中...' : '開始遊戲'}
            </button>
          </div>
        )}
      </div>

      {/* Invite Code */}
      <div className="flex flex-col gap-2">
        <h2 className="text-sm font-semibold text-text-secondary">
          邀請碼
        </h2>
        <div className="flex items-center gap-3">
          <span className="font-mono text-2xl tracking-widest text-gold">
            {session.inviteCode}
          </span>
          <button
            className="rounded border border-border px-3 py-1 text-xs text-text-secondary transition-colors hover:text-text-primary"
            onClick={() =>
              navigator.clipboard
                .writeText(session.inviteCode)
                .then(() =>
                  useToastStore.getState().addToast('邀請碼已複製', 'success'),
                )
                .catch(() =>
                  useToastStore.getState().addToast('複製失敗', 'error'),
                )
            }
          >
            複製
          </button>
        </div>
        <p className="text-xs text-text-tertiary">
          將此邀請碼分享給玩家，讓他們加入場次。
        </p>
      </div>

      {/* Players */}
      <div className="flex flex-col gap-3">
        <h2 className="text-sm font-semibold text-text-secondary">
          玩家
        </h2>
        <SessionPlayerList sessionId={session.id} isGm={isGm} />
      </div>

      {/* Player: Character Selection (lobby only) */}
      {!isGm && session.status === 'lobby' && (
        <div className="flex flex-col gap-3">
          <h2 className="text-sm font-semibold text-text-secondary">
            你的角色
          </h2>
          {assignedName ? (
            <p className="text-sm text-green-500">
              已分配角色：{assignedName}
            </p>
          ) : (
            <div className="flex items-center gap-3">
              <select
                className="rounded-lg border border-border bg-bg-input px-3 py-2.5 text-sm text-text-primary outline-none"
                value={selectedCharId}
                onChange={(e) => setSelectedCharId(e.target.value)}
              >
                <option value="">
                  {characters.length === 0
                    ? '沒有角色 — 請先建立一個'
                    : '選擇角色'}
                </option>
                {characters.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </select>
              <Button
                size="sm"
                onClick={handleAssignCharacter}
                loading={assignLoading}
                disabled={!selectedCharId}
              >
                分配角色
              </Button>
              <Link
                to={ROUTES.CHARACTERS}
                className="text-xs text-gold hover:text-gold-light"
              >
                + 新建角色
              </Link>
            </div>
          )}
        </div>
      )}

      {/* Player waiting message */}
      {!isGm && session.status === 'lobby' && (
        <p className="text-center text-sm text-text-tertiary">
          等待 GM 開始遊戲...
        </p>
      )}
    </div>
  )
}
