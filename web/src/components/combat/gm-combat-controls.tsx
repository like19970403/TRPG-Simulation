import { useState } from 'react'
import { cn } from '../../lib/cn'

interface PlayerInfo {
  userId: string
  name: string
  ready: boolean
  actionLabel?: string
}

interface GmCombatControlsProps {
  players: PlayerInfo[]
  totalPlayers: number
  enemyName: string
  onEnemyAction: (action: { type: string; target: string }) => void
  onExecuteRound: () => void
  onEndCombat: () => void
  executing: boolean
}

export function GmCombatControls({
  players,
  totalPlayers,
  enemyName,
  onEnemyAction,
  onExecuteRound,
  onEndCombat,
  executing,
}: GmCombatControlsProps) {
  const [enemyMode, setEnemyMode] = useState<'manual' | 'auto'>('manual')
  const [enemyActionType, setEnemyActionType] = useState('attack')
  const [enemyTarget, setEnemyTarget] = useState(players[0]?.userId ?? '')
  const [enemyReady, setEnemyReady] = useState(false)

  const readyCount = players.filter((p) => p.ready).length
  const allReady = readyCount === totalPlayers && (enemyReady || enemyMode === 'auto')

  const handleEnemyConfirm = () => {
    onEnemyAction({ type: enemyActionType, target: enemyTarget })
    setEnemyReady(true)
  }

  const handleExecute = () => {
    if (enemyMode === 'auto') {
      // Random target
      const alive = players.filter((p) => p.userId)
      const target = alive[Math.floor(Math.random() * alive.length)]
      onEnemyAction({ type: 'attack', target: target?.userId ?? '' })
    }
    onExecuteRound()
    setEnemyReady(false)
  }

  return (
    <div className="flex flex-col gap-3">
      {/* Enemy action area */}
      <div className="flex flex-col gap-2 bg-bg-sidebar p-3">
        <div className="flex items-center gap-3">
          <span className="text-xs font-semibold text-text-secondary">敵人行動</span>
          <div className="flex gap-1">
            {(['manual', 'auto'] as const).map((mode) => (
              <button
                key={mode}
                type="button"
                onClick={() => { setEnemyMode(mode); setEnemyReady(false) }}
                className={cn(
                  'px-2.5 py-1 text-[10px] font-medium transition-colors',
                  enemyMode === mode
                    ? 'bg-gold text-bg-page'
                    : 'bg-border text-text-tertiary hover:text-text-secondary',
                )}
              >
                {mode === 'manual' ? '手動' : '自動'}
              </button>
            ))}
          </div>
        </div>

        {enemyMode === 'manual' ? (
          <div className="flex items-center gap-2">
            <select
              value={enemyActionType}
              onChange={(e) => setEnemyActionType(e.target.value)}
              className="rounded-none border border-border bg-border px-3 py-1.5 text-[11px] text-text-primary outline-none"
            >
              <option value="attack">攻擊</option>
              <option value="defend">防禦</option>
            </select>
            <span className="text-[11px] text-text-tertiary">→</span>
            <select
              value={enemyTarget}
              onChange={(e) => setEnemyTarget(e.target.value)}
              className="rounded-none border border-border bg-border px-3 py-1.5 text-[11px] text-text-primary outline-none"
            >
              {players.map((p) => (
                <option key={p.userId} value={p.userId}>{p.name}</option>
              ))}
            </select>
            <button
              type="button"
              onClick={handleEnemyConfirm}
              disabled={enemyReady}
              className="rounded-none bg-gold px-3 py-1.5 text-[11px] font-semibold text-bg-page disabled:opacity-40"
            >
              {enemyReady ? '✓' : '確定'}
            </button>
          </div>
        ) : (
          <span className="text-[10px] text-text-tertiary">
            {enemyName} 將隨機選擇行動和目標
          </span>
        )}
      </div>

      {/* Player status */}
      <div className="flex flex-col gap-1.5">
        {players.map((p) => (
          <div key={p.userId} className="flex items-center justify-between">
            <span className="text-[11px] text-text-primary">{p.name}</span>
            {p.ready ? (
              <span className="text-[11px] font-medium text-emerald-400">✓ {p.actionLabel}</span>
            ) : (
              <span className="text-[11px] text-text-tertiary">... 思考中</span>
            )}
          </div>
        ))}
      </div>

      {/* Execute / End buttons */}
      <div className="flex items-center justify-between">
        <span className="text-xs text-text-tertiary">
          全員就緒: {readyCount}/{totalPlayers}
        </span>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={onEndCombat}
            className="rounded-none border border-error px-4 py-2 text-xs font-medium text-error transition-colors hover:bg-error/10"
          >
            結束戰鬥
          </button>
          <button
            type="button"
            onClick={handleExecute}
            disabled={!allReady || executing}
            className="rounded-none bg-gold px-6 py-2 text-xs font-semibold text-bg-page transition-colors disabled:opacity-40"
          >
            {executing ? '結算中...' : '執行回合'}
          </button>
        </div>
      </div>
    </div>
  )
}
