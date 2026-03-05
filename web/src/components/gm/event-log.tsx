import { useEffect, useRef } from 'react'
import { useGameStore } from '../../stores/game-store'
import { cn } from '../../lib/cn'

const eventTypeColors: Record<string, string> = {
  state_sync: 'bg-blue-500/20 text-blue-400',
  scene_changed: 'bg-purple-500/20 text-purple-400',
  dice_rolled: 'bg-yellow-500/20 text-yellow-400',
  item_revealed: 'bg-green-500/20 text-green-400',
  item_given: 'bg-green-500/20 text-green-400',
  item_removed: 'bg-red-500/20 text-red-400',
  npc_field_revealed: 'bg-green-500/20 text-green-400',
  variable_changed: 'bg-cyan-500/20 text-cyan-400',
  player_choice: 'bg-orange-500/20 text-orange-400',
  gm_broadcast: 'bg-gold/20 text-gold',
  game_paused: 'bg-yellow-500/20 text-yellow-400',
  game_resumed: 'bg-green-500/20 text-green-400',
  game_ended: 'bg-red-500/20 text-red-400',
  error: 'bg-error/20 text-error',
}

function formatTime(timestamp: number): string {
  const date = new Date(timestamp * 1000)
  return date.toLocaleTimeString('zh-TW', {
    hour12: false,
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

function summarizePayload(type: string, payload: unknown): string {
  const p = payload as Record<string, unknown>
  switch (type) {
    case 'scene_changed':
      return `場景 → ${p.scene_id}`
    case 'dice_rolled':
      return `${p.formula} = ${p.total}`
    case 'item_revealed':
      return `道具 ${p.item_id} 已揭露`
    case 'item_given':
      return `道具「${p.item_id}」已給予${p.quantity && Number(p.quantity) > 1 ? ` x${p.quantity}` : ''}`
    case 'item_removed':
      return `道具「${p.item_id}」已移除${p.quantity && Number(p.quantity) > 1 ? ` x${p.quantity}` : ''}`
    case 'npc_field_revealed':
      return `NPC ${p.npc_id} 欄位「${p.field_key}」已揭露`
    case 'variable_changed':
      return `${p.name}: ${JSON.stringify(p.old_value)} → ${JSON.stringify(p.new_value)}`
    case 'player_choice':
      return `投了一票給「${p.transition_label ?? `轉換 #${p.transition_index}`}」`
    case 'gm_broadcast':
      return p.content ? String(p.content).slice(0, 60) : '（圖片）'
    case 'game_paused':
      return '遊戲已暫停'
    case 'game_resumed':
      return '遊戲已繼續'
    case 'game_ended':
      return '遊戲已結束'
    case 'state_sync':
      return '狀態已同步'
    case 'error':
      return p.message ? String(p.message) : '發生錯誤'
    default:
      return type
  }
}

export function EventLog() {
  const eventLog = useGameStore((s) => s.eventLog)
  const scrollRef = useRef<HTMLDivElement>(null)
  const isNearBottomRef = useRef(true)

  useEffect(() => {
    const el = scrollRef.current
    if (!el) return
    const handleScroll = () => {
      isNearBottomRef.current =
        el.scrollHeight - el.scrollTop - el.clientHeight < 60
    }
    el.addEventListener('scroll', handleScroll, { passive: true })
    return () => el.removeEventListener('scroll', handleScroll)
  }, [])

  useEffect(() => {
    if (isNearBottomRef.current && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [eventLog.length])

  return (
    <div ref={scrollRef} className="flex-1 overflow-y-auto p-4">
      {eventLog.length === 0 ? (
        <p className="text-xs text-text-tertiary">尚無事件</p>
      ) : (
        <div className="flex flex-col gap-1">
          {eventLog.map((entry) => (
            <div
              key={entry.id}
              className="flex items-start gap-2 text-xs"
            >
              <span className="shrink-0 font-mono text-text-tertiary">
                {formatTime(entry.timestamp)}
              </span>
              <span
                className={cn(
                  'shrink-0 rounded px-1.5 py-0.5 font-mono text-[10px]',
                  eventTypeColors[entry.type] ??
                    'bg-text-tertiary/20 text-text-tertiary',
                )}
              >
                {entry.type}
              </span>
              <span className="text-text-secondary">
                {summarizePayload(entry.type, entry.payload)}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
