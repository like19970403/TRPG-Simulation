import { useEffect, useRef } from 'react'

export interface CombatLogEntry {
  id: string
  text: string
  type: 'action' | 'result' | 'damage' | 'info'
}

interface CombatLogProps {
  entries: CombatLogEntry[]
}

const TYPE_COLORS: Record<CombatLogEntry['type'], string> = {
  action: 'text-text-primary',
  result: 'text-text-secondary',
  damage: 'text-error font-medium',
  info: 'text-gold',
}

export function CombatLog({ entries }: CombatLogProps) {
  const endRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [entries.length])

  return (
    <div className="flex flex-1 flex-col gap-1 overflow-y-auto bg-bg-sidebar p-3">
      <span className="text-[10px] font-semibold uppercase tracking-wider text-text-tertiary">
        戰鬥日誌
      </span>
      {entries.length === 0 && (
        <span className="text-[10px] text-text-tertiary">等待戰鬥開始...</span>
      )}
      {entries.map((entry) => (
        <p key={entry.id} className={`text-[11px] ${TYPE_COLORS[entry.type]}`}>
          {entry.text}
        </p>
      ))}
      <div ref={endRef} />
    </div>
  )
}
