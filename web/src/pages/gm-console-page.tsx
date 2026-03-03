import { useState } from 'react'
import { useParams, useNavigate } from 'react-router'
import { useGameSocket } from '../hooks/use-game-socket'
import { useGameStore } from '../stores/game-store'
import { pauseSession, resumeSession, endSession } from '../api/sessions'
import { ROUTES } from '../lib/constants'
import { GmTopBar } from '../components/gm/gm-top-bar'
import { PlayerPanel } from '../components/gm/player-panel'
import { ScenePanel } from '../components/gm/scene-panel'
import { ItemsPanel } from '../components/gm/items-panel'
import { EventLog } from '../components/gm/event-log'
import { DiceLog } from '../components/gm/dice-log'
import { BroadcastPanel } from '../components/gm/broadcast-panel'
import { VariablesPanel } from '../components/gm/variables-panel'
import { NotesPanel } from '../components/ui/notes-panel'
import { GameStatusOverlay } from '../components/player/game-status-overlay'
import { LoadingSpinner } from '../components/ui/loading-spinner'
import { cn } from '../lib/cn'

type BottomTab = 'events' | 'dice' | 'broadcast' | 'variables' | 'notes'

export function GmConsolePage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { sendAction, connectionStatus, error } = useGameSocket(id!)
  const [activeTab, setActiveTab] = useState<BottomTab>('events')

  const session = useGameStore((s) => s.session)
  const scenarioContent = useGameStore((s) => s.scenarioContent)
  const gameState = useGameStore((s) => s.gameState)

  // Loading state
  if (!gameState) {
    return (
      <div className="flex h-screen flex-col items-center justify-center gap-4 bg-bg-page">
        {error ? (
          <div className="text-center">
            <p className="text-error">{error}</p>
            <p className="mt-2 text-sm text-text-tertiary">
              無法連線至遊戲場次
            </p>
          </div>
        ) : (
          <>
            <LoadingSpinner className="h-8 w-8 text-gold" />
            <p className="text-sm text-text-tertiary">
              正在連線至遊戲場次...
            </p>
          </>
        )}
      </div>
    )
  }

  const scenarioTitle = scenarioContent?.title ?? '未命名劇本'
  const sessionStatus = (gameState.status ?? session?.status ?? 'active') as
    | 'lobby'
    | 'active'
    | 'paused'
    | 'completed'

  const tabs: { key: BottomTab; label: string }[] = [
    { key: 'events', label: '事件' },
    { key: 'dice', label: '骰子紀錄' },
    { key: 'broadcast', label: '廣播' },
    { key: 'variables', label: '變數' },
    { key: 'notes', label: '筆記' },
  ]

  return (
    <div className="flex h-screen flex-col bg-bg-page">
      {/* Connection status banner */}
      {connectionStatus === 'reconnecting' && (
        <div className="bg-yellow-600/20 px-4 py-1.5 text-center text-xs text-yellow-400">
          正在重新連線至遊戲伺服器...
        </div>
      )}

      {/* Top bar */}
      <GmTopBar
        scenarioTitle={scenarioTitle}
        status={sessionStatus}
        onPause={() => {
          if (id) pauseSession(id)
        }}
        onResume={() => {
          if (id) resumeSession(id)
        }}
        onEnd={async () => {
          if (id) {
            await endSession(id)
            navigate(ROUTES.SESSIONS)
          }
        }}
      />

      <div className="h-px bg-border" />

      {/* Main 3-column area */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left: Players */}
        <PlayerPanel />
        <div className="w-px bg-border" />

        {/* Center: Scene */}
        <ScenePanel sendAction={sendAction} />
        <div className="w-px bg-border" />

        {/* Right: Items & NPCs */}
        <ItemsPanel sendAction={sendAction} />
      </div>

      <div className="h-px bg-border" />

      {/* Bottom bar */}
      <div className="flex h-[180px] flex-col bg-bg-sidebar">
        {/* Tab bar */}
        <div className="flex border-b border-border">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              className={cn(
                'px-4 py-2 text-xs font-medium transition-colors',
                activeTab === tab.key
                  ? 'border-b-2 border-gold text-gold'
                  : 'text-text-tertiary hover:text-text-secondary',
              )}
              onClick={() => setActiveTab(tab.key)}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {/* Tab content */}
        {activeTab === 'events' && <EventLog />}
        {activeTab === 'dice' && <DiceLog sendAction={sendAction} />}
        {activeTab === 'broadcast' && (
          <BroadcastPanel sendAction={sendAction} />
        )}
        {activeTab === 'variables' && (
          <VariablesPanel sendAction={sendAction} />
        )}
        {activeTab === 'notes' && id && <NotesPanel sessionId={id} />}
      </div>

      {/* Game ended overlay (safety net for WS-driven end) */}
      <GameStatusOverlay isGm />
    </div>
  )
}
