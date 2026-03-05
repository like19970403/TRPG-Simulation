import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useParams, Link } from 'react-router'
import { listSessionEvents, getSession } from '../api/sessions'
import { getScenario } from '../api/scenarios'
import type { ReplayEvent, ScenarioContent } from '../api/types'
import { ROUTES } from '../lib/constants'
import { Button } from '../components/ui/button'
import { LoadingSpinner } from '../components/ui/loading-spinner'
import { Markdown } from '../components/ui/markdown'
import { cn } from '../lib/cn'

const EVENT_COLORS: Record<string, string> = {
  game_started: 'bg-green-500/20 text-green-400',
  scene_changed: 'bg-purple-500/20 text-purple-400',
  dice_rolled: 'bg-yellow-500/20 text-yellow-400',
  item_revealed: 'bg-green-500/20 text-green-400',
  item_given: 'bg-green-500/20 text-green-400',
  item_removed: 'bg-red-500/20 text-red-400',
  npc_field_revealed: 'bg-green-500/20 text-green-400',
  variable_changed: 'bg-cyan-500/20 text-cyan-400',
  player_choice: 'bg-orange-500/20 text-orange-400',
  player_votes: 'bg-orange-500/20 text-orange-400',
  gm_broadcast: 'bg-gold/20 text-gold',
  game_paused: 'bg-yellow-500/20 text-yellow-400',
  game_resumed: 'bg-green-500/20 text-green-400',
  game_ended: 'bg-red-500/20 text-red-400',
  player_joined: 'bg-blue-500/20 text-blue-400',
  player_left: 'bg-gray-500/20 text-gray-400',
}

function summarize(type: string, payload: unknown): string {
  const p = payload as Record<string, unknown>
  switch (type) {
    case 'game_started':
      return '遊戲開始'
    case 'scene_changed':
      return `場景 → ${p.scene_id}`
    case 'dice_rolled':
      return `${p.formula} = ${p.total}`
    case 'item_given':
      return `道具「${p.item_id}」已給予${p.quantity && Number(p.quantity) > 1 ? ` x${p.quantity}` : ''}`
    case 'item_removed':
      return `道具「${p.item_id}」已移除`
    case 'item_revealed':
      return `道具「${p.item_id}」已揭露`
    case 'npc_field_revealed':
      return `NPC ${p.npc_id} 欄位「${p.field_key}」已揭露`
    case 'variable_changed':
      return `${p.name}: ${JSON.stringify(p.old_value)} → ${JSON.stringify(p.new_value)}`
    case 'player_choice':
      return `投票：轉換 #${p.transition_index}`
    case 'gm_broadcast':
      return p.content ? String(p.content).slice(0, 80) : '（圖片）'
    case 'game_paused':
      return '遊戲暫停'
    case 'game_resumed':
      return '遊戲繼續'
    case 'game_ended':
      return '遊戲結束'
    case 'player_joined':
      return `玩家加入：${p.username ?? p.user_id}`
    case 'player_left':
      return `玩家離線：${p.username ?? p.user_id}`
    default:
      return type
  }
}

function formatTimestamp(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleTimeString('zh-TW', {
    hour12: false,
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

/** Duration between first and last event in readable format */
function formatDuration(events: ReplayEvent[]): string {
  if (events.length < 2) return '--'
  const start = new Date(events[0].createdAt).getTime()
  const end = new Date(events[events.length - 1].createdAt).getTime()
  const totalSec = Math.round((end - start) / 1000)
  const h = Math.floor(totalSec / 3600)
  const m = Math.floor((totalSec % 3600) / 60)
  const s = totalSec % 60
  if (h > 0) return `${h}h ${m}m`
  if (m > 0) return `${m}m ${s}s`
  return `${s}s`
}

type PlaybackSpeed = 1 | 2 | 4 | 8

export function SessionReplayPage() {
  const { id } = useParams<{ id: string }>()

  const [events, setEvents] = useState<ReplayEvent[]>([])
  const [scenarioContent, setScenarioContent] = useState<ScenarioContent | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  // Playback state
  const [cursor, setCursor] = useState(0)
  const [playing, setPlaying] = useState(false)
  const [speed, setSpeed] = useState<PlaybackSpeed>(1)

  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const listRef = useRef<HTMLDivElement>(null)

  // Load events and scenario content
  useEffect(() => {
    if (!id) return
    setLoading(true)
    Promise.all([listSessionEvents(id), getSession(id)])
      .then(async ([evts, session]) => {
        setEvents(evts)
        if (session.scenarioId) {
          try {
            const sc = await getScenario(session.scenarioId)
            if (sc.content) {
              setScenarioContent(
                typeof sc.content === 'string'
                  ? (JSON.parse(sc.content) as ScenarioContent)
                  : (sc.content as unknown as ScenarioContent),
              )
            }
          } catch {
            // Scenario may have been deleted — still show events
          }
        }
      })
      .catch((err) => setError(err instanceof Error ? err.message : '載入失敗'))
      .finally(() => setLoading(false))
  }, [id])

  // Auto-scroll to active event
  useEffect(() => {
    if (!listRef.current) return
    const activeEl = listRef.current.querySelector('[data-active="true"]')
    if (activeEl) {
      activeEl.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
    }
  }, [cursor])

  // Playback timer
  const advanceCursor = useCallback(() => {
    setCursor((prev) => {
      if (prev >= events.length - 1) {
        setPlaying(false)
        return prev
      }
      return prev + 1
    })
  }, [events.length])

  useEffect(() => {
    if (!playing || events.length === 0) return

    // Compute delay based on real time between events, scaled by speed
    const currentEvt = events[cursor]
    const nextEvt = events[cursor + 1]
    if (!nextEvt) {
      setPlaying(false)
      return
    }

    const realDelayMs =
      new Date(nextEvt.createdAt).getTime() -
      new Date(currentEvt.createdAt).getTime()
    // Clamp between 100ms and 3000ms after speed adjustment
    const scaledDelay = Math.min(3000, Math.max(100, realDelayMs / speed))

    timerRef.current = setTimeout(advanceCursor, scaledDelay)
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current)
    }
  }, [playing, cursor, speed, events, advanceCursor])

  // Current scene derived from events up to cursor
  const currentScene = useMemo(() => {
    for (let i = cursor; i >= 0; i--) {
      if (events[i]?.type === 'scene_changed') {
        return (events[i].payload as Record<string, unknown>).scene_id as string
      }
    }
    return scenarioContent?.start_scene ?? null
  }, [cursor, events, scenarioContent])

  const scene = scenarioContent?.scenes?.find((s) => s.id === currentScene)

  if (loading) {
    return (
      <div className="flex h-screen items-center justify-center bg-bg-page">
        <LoadingSpinner className="h-8 w-8 text-gold" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex h-screen flex-col items-center justify-center gap-4 bg-bg-page">
        <p className="text-error">{error}</p>
        <Link
          to={ROUTES.SESSIONS}
          className="text-sm text-gold hover:underline"
        >
          返回場次列表
        </Link>
      </div>
    )
  }

  return (
    <div className="flex h-screen flex-col bg-bg-page">
      {/* Top bar */}
      <div className="flex h-14 items-center justify-between border-b border-border bg-bg-sidebar px-6">
        <div className="flex items-center gap-3">
          <Link
            to={ROUTES.SESSIONS}
            className="text-sm text-text-tertiary hover:text-gold"
          >
            ← 場次列表
          </Link>
          <div className="h-5 w-px bg-border" />
          <span className="font-display text-lg font-bold text-gold">
            遊戲回放
          </span>
          <span className="text-xs text-text-tertiary">
            {events.length} 事件 · {formatDuration(events)}
          </span>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-text-tertiary">
            {cursor + 1} / {events.length}
          </span>
        </div>
      </div>

      {/* Main content */}
      <div className="flex flex-1 flex-col overflow-hidden md:flex-row">
        {/* Left: Scene preview */}
        <div className="flex flex-1 flex-col overflow-y-auto p-4 md:p-6">
          {scene ? (
            <div className="mx-auto w-full max-w-2xl">
              <div className="rounded-xl border border-gold/30 bg-parchment p-8">
                <h2 className="mb-4 font-display text-2xl font-bold text-text-primary">
                  {scene.name}
                </h2>
                <Markdown className="text-sm leading-relaxed text-text-secondary">
                  {scene.content}
                </Markdown>
              </div>
            </div>
          ) : (
            <div className="flex flex-1 items-center justify-center">
              <p className="text-sm text-text-tertiary">
                {scenarioContent ? '等待場景開始...' : '無劇本資料'}
              </p>
            </div>
          )}
        </div>

        {/* Right: Event timeline */}
        <div className="flex w-full flex-col border-t border-border bg-bg-sidebar md:w-[380px] md:border-l md:border-t-0">
          <div
            ref={listRef}
            className="flex-1 overflow-y-auto p-4"
          >
            {events.length === 0 ? (
              <p className="text-xs text-text-tertiary">此場次無事件</p>
            ) : (
              <div className="flex flex-col gap-0.5">
                {events.map((evt, i) => (
                  <button
                    key={evt.id}
                    data-active={i === cursor}
                    className={cn(
                      'flex items-start gap-2 rounded px-2 py-1 text-left text-xs transition-colors',
                      i === cursor
                        ? 'bg-gold/10 ring-1 ring-gold/30'
                        : i < cursor
                          ? 'opacity-60'
                          : 'opacity-40',
                      'hover:bg-bg-input',
                    )}
                    onClick={() => {
                      setCursor(i)
                      setPlaying(false)
                    }}
                  >
                    <span className="shrink-0 font-mono text-text-tertiary">
                      {formatTimestamp(evt.createdAt)}
                    </span>
                    <span
                      className={cn(
                        'shrink-0 rounded px-1.5 py-0.5 font-mono text-[10px]',
                        EVENT_COLORS[evt.type] ??
                          'bg-text-tertiary/20 text-text-tertiary',
                      )}
                    >
                      {evt.type}
                    </span>
                    <span className="truncate text-text-secondary">
                      {summarize(evt.type, evt.payload)}
                    </span>
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Bottom: Playback controls */}
      <div className="flex h-14 items-center justify-center gap-2 border-t border-border bg-bg-sidebar px-3 md:gap-4 md:px-6">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => {
            setCursor(0)
            setPlaying(false)
          }}
          disabled={cursor === 0}
        >
          ⏮
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => {
            setCursor((c) => Math.max(0, c - 1))
            setPlaying(false)
          }}
          disabled={cursor === 0}
        >
          ◀
        </Button>
        <Button
          variant="primary"
          size="sm"
          onClick={() => {
            if (cursor >= events.length - 1) {
              setCursor(0)
            }
            setPlaying((p) => !p)
          }}
          className="w-20"
        >
          {playing ? '暫停' : '播放'}
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => {
            setCursor((c) => Math.min(events.length - 1, c + 1))
            setPlaying(false)
          }}
          disabled={cursor >= events.length - 1}
        >
          ▶
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => {
            setCursor(events.length - 1)
            setPlaying(false)
          }}
          disabled={cursor >= events.length - 1}
        >
          ⏭
        </Button>

        <div className="h-5 w-px bg-border" />

        {/* Speed selector */}
        <div className="flex items-center gap-1">
          {([1, 2, 4, 8] as PlaybackSpeed[]).map((s) => (
            <button
              key={s}
              className={cn(
                'rounded px-2 py-1 text-xs transition-colors',
                speed === s
                  ? 'bg-gold/20 text-gold'
                  : 'text-text-tertiary hover:text-text-secondary',
              )}
              onClick={() => setSpeed(s)}
            >
              {s}x
            </button>
          ))}
        </div>

        <div className="h-5 w-px bg-border" />

        {/* Timeline slider */}
        <input
          type="range"
          min={0}
          max={Math.max(0, events.length - 1)}
          value={cursor}
          onChange={(e) => {
            setCursor(Number(e.target.value))
            setPlaying(false)
          }}
          className="w-24 accent-gold md:w-48"
        />
      </div>
    </div>
  )
}
