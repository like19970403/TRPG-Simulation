import { cn } from '../../lib/cn'

interface HpBarProps {
  current: number
  max: number
  height?: string
  showLabel?: boolean
}

export function HpBar({ current, max, height = 'h-3', showLabel = true }: HpBarProps) {
  const pct = max > 0 ? Math.max(0, Math.min(100, (current / max) * 100)) : 0
  const color =
    pct > 50
      ? 'bg-emerald-500'
      : pct > 25
        ? 'bg-amber-500'
        : 'bg-red-500'

  return (
    <div className="flex items-center gap-2 w-full">
      <div className={cn('flex-1 rounded-none overflow-hidden bg-border', height)}>
        <div
          className={cn('h-full transition-all duration-500', color)}
          style={{ width: `${pct}%` }}
        />
      </div>
      {showLabel && (
        <span className="shrink-0 text-[10px] font-medium text-text-tertiary">
          {current}/{max}
        </span>
      )}
    </div>
  )
}
