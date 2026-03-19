import { useEffect, useState } from 'react'
import { listSessionPlayers, removeSessionPlayer } from '../../api/sessions'
import { parseProfile } from '../../lib/character-profile'
import { RULE_PRESETS } from '../../data/rule-presets'
import type { SessionPlayerResponse } from '../../api/types'

interface SessionPlayerListProps {
  sessionId: string
  isGm?: boolean
}

const POLL_INTERVAL_MS = 3000

function getSkillNames(notes: string | undefined) {
  if (!notes) return null
  const profile = parseProfile(notes)
  if (!profile) return null

  const preset = RULE_PRESETS.find((p) => p.id === profile._system)
  if (!preset) return null

  const skills = (profile._startingSkills as string[] | undefined) ?? []
  const cultivation = profile._startingCultivation as string | undefined

  const skillNames = skills
    .map((id) => preset.martialSkills?.find((s) => s.id === id)?.name)
    .filter(Boolean)
  const cultivationName = preset.cultivationMethods?.find(
    (c) => c.id === cultivation,
  )?.name

  if (skillNames.length === 0 && !cultivationName) return null
  return { skillNames, cultivationName }
}

export function SessionPlayerList({ sessionId, isGm }: SessionPlayerListProps) {
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

  async function handleRemove(player: SessionPlayerResponse) {
    if (!confirm(`確定要移除玩家 ${player.username ?? player.userId.slice(0, 8)}？`)) return
    try {
      await removeSessionPlayer(sessionId, player.userId)
      setPlayers((prev) => prev.filter((p) => p.id !== player.id))
    } catch {
      // Will be refreshed on next poll
    }
  }

  if (loading) {
    return (
      <p className="text-sm text-text-tertiary">載入玩家中...</p>
    )
  }

  if (players.length === 0) {
    return (
      <p className="text-sm text-text-tertiary">
        尚未有玩家加入。
      </p>
    )
  }

  return (
    <ul className="flex flex-col gap-2">
      {players.map((player) => {
        const info = getSkillNames(player.characterNotes)
        return (
          <li
            key={player.id}
            className="flex items-center justify-between rounded-md border border-border bg-bg-card px-4 py-2.5"
          >
            <div className="flex flex-col gap-0.5">
              <span className="text-sm text-text-primary">
                {player.username ?? `玩家 ${player.userId.slice(0, 8)}`}
              </span>
              {player.characterName && (
                <span className="text-xs text-gold">
                  角色：{player.characterName}
                </span>
              )}
              {info && (
                <div className="flex flex-wrap gap-1.5 mt-0.5">
                  {info.skillNames.map((name) => (
                    <span
                      key={name}
                      className="rounded bg-emerald-900/40 px-1.5 py-0.5 text-[9px] text-emerald-400"
                    >
                      {name}
                    </span>
                  ))}
                  {info.cultivationName && (
                    <span className="rounded bg-amber-900/40 px-1.5 py-0.5 text-[9px] text-amber-400">
                      {info.cultivationName}
                    </span>
                  )}
                </div>
              )}
            </div>
            <div className="flex items-center gap-3">
              <span className="text-xs text-text-tertiary">
                {player.characterId ? '已分配' : '未分配角色'}
              </span>
              {isGm && (
                <button
                  className="text-xs text-error transition-colors hover:text-error/80 cursor-pointer"
                  onClick={() => handleRemove(player)}
                  title="移除玩家"
                >
                  ✕
                </button>
              )}
            </div>
          </li>
        )
      })}
    </ul>
  )
}
