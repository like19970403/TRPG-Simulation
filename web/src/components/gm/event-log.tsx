import { useEffect, useRef } from 'react'
import { useGameStore } from '../../stores/game-store'
import { cn } from '../../lib/cn'

const eventTypeColors: Record<string, string> = {
  state_sync: 'bg-blue-500/20 text-blue-400',
  scene_changed: 'bg-purple-500/20 text-purple-400',
  dice_rolled: 'bg-yellow-500/20 text-yellow-400',
  item_revealed: 'bg-green-500/20 text-green-400',
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
  return date.toLocaleTimeString('en-US', {
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
      return `Scene → ${p.scene_id}`
    case 'dice_rolled':
      return `${p.formula} = ${p.total}`
    case 'item_revealed':
      return `Item ${p.item_id} revealed`
    case 'npc_field_revealed':
      return `NPC ${p.npc_id} field "${p.field_key}" revealed`
    case 'variable_changed':
      return `${p.name}: ${JSON.stringify(p.old_value)} → ${JSON.stringify(p.new_value)}`
    case 'player_choice':
      return `Player chose transition ${p.transition_index}`
    case 'gm_broadcast':
      return p.content ? String(p.content).slice(0, 60) : '(image)'
    case 'game_paused':
      return 'Game paused'
    case 'game_resumed':
      return 'Game resumed'
    case 'game_ended':
      return 'Game ended'
    case 'state_sync':
      return 'State synchronized'
    case 'error':
      return p.message ? String(p.message) : 'Error occurred'
    default:
      return type
  }
}

export function EventLog() {
  const eventLog = useGameStore((s) => s.eventLog)
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [eventLog.length])

  return (
    <div ref={scrollRef} className="flex-1 overflow-y-auto p-4">
      {eventLog.length === 0 ? (
        <p className="text-xs text-text-tertiary">No events yet</p>
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
