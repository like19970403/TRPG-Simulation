import { useState } from 'react'
import { useParams, useNavigate } from 'react-router'
import { useGameSocket } from '../hooks/use-game-socket'
import { useGameStore } from '../stores/game-store'
import { useKeyboardShortcuts } from '../hooks/use-keyboard-shortcuts'
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
import { ConnectionIndicator } from '../components/connection-indicator'

type BottomTab = 'events' | 'dice' | 'broadcast' | 'variables' | 'notes'
type MobilePanel = 'scene' | 'players' | 'items'

export function GmConsolePage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { sendAction, connectionStatus, error } = useGameSocket(id!)
  const [activeTab, setActiveTab] = useState<BottomTab>('events')
  const [mobilePanel, setMobilePanel] = useState<MobilePanel>('scene')

  const session = useGameStore((s) => s.session)
  const scenarioContent = useGameStore((s) => s.scenarioContent)
  const gameState = useGameStore((s) => s.gameState)

  const tabKeys: BottomTab[] = ['events', 'dice', 'broadcast', 'variables', 'notes']

  useKeyboardShortcuts(
    [
      // Ctrl+1~5: switch bottom tabs
      { key: 'Digit1', ctrl: true, handler: () => setActiveTab(tabKeys[0]) },
      { key: 'Digit2', ctrl: true, handler: () => setActiveTab(tabKeys[1]) },
      { key: 'Digit3', ctrl: true, handler: () => setActiveTab(tabKeys[2]) },
      { key: 'Digit4', ctrl: true, handler: () => setActiveTab(tabKeys[3]) },
      { key: 'Digit5', ctrl: true, handler: () => setActiveTab(tabKeys[4]) },
      // Ctrl+P: toggle pause/resume
      {
        key: 'KeyP',
        ctrl: true,
        handler: () => {
          if (!id) return
          const status = gameState?.status ?? session?.status
          if (status === 'active') pauseSession(id)
          else if (status === 'paused') resumeSession(id)
        },
      },
    ],
    [id, gameState?.status, session?.status],
  )

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

  const tabs: { key: BottomTab; label: string; shortcut: string; tip: string }[] = [
    { key: 'events', label: '事件', shortcut: '⌃1', tip: '所有遊戲事件的即時日誌' },
    { key: 'dice', label: '骰子紀錄', shortcut: '⌃2', tip: '擲骰介面與歷史紀錄，可設定用途說明' },
    { key: 'broadcast', label: '廣播', shortcut: '⌃3', tip: '向全體玩家推送文字或圖片訊息' },
    { key: 'variables', label: '變數', shortcut: '⌃4', tip: '劇本全域變數，可即時編輯影響場景條件' },
    { key: 'notes', label: '筆記', shortcut: '⌃5', tip: '此場次的私人筆記，僅自己可見' },
  ]

  return (
    <div className="flex h-screen flex-col bg-bg-page">
      <ConnectionIndicator status={connectionStatus} />

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

      {/* Mobile panel switcher */}
      <div className="flex border-b border-border lg:hidden">
        {([
          { key: 'scene', label: '場景' },
          { key: 'players', label: '玩家' },
          { key: 'items', label: '道具/NPC' },
        ] as { key: MobilePanel; label: string }[]).map((p) => (
          <button
            key={p.key}
            className={cn(
              'flex-1 px-3 py-2 text-xs font-medium transition-colors',
              mobilePanel === p.key
                ? 'border-b-2 border-gold text-gold'
                : 'text-text-tertiary hover:text-text-secondary',
            )}
            onClick={() => setMobilePanel(p.key)}
          >
            {p.label}
          </button>
        ))}
      </div>

      {/* Main 3-column area — desktop: 3-col, tablet/mobile: switched via tabs */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left: Players */}
        <div className={cn('overflow-y-auto lg:block', mobilePanel === 'players' ? 'block w-full' : 'hidden')}>
          <PlayerPanel />
        </div>
        <div className="hidden w-px bg-border lg:block" />

        {/* Center: Scene */}
        <div className={cn('flex-1 overflow-y-auto lg:block', mobilePanel === 'scene' ? 'block' : 'hidden')}>
          <ScenePanel sendAction={sendAction} />
        </div>
        <div className="hidden w-px bg-border lg:block" />

        {/* Right: Items & NPCs */}
        <div className={cn('overflow-y-auto lg:block', mobilePanel === 'items' ? 'block w-full' : 'hidden')}>
          <ItemsPanel sendAction={sendAction} />
        </div>
      </div>

      <div className="h-px bg-border" />

      {/* Bottom bar */}
      <div className="flex h-36 flex-col bg-bg-sidebar lg:h-45">
        {/* Tab bar */}
        <div className="flex border-b border-border">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              title={tab.tip}
              className={cn(
                'px-4 py-2 text-xs font-medium transition-colors',
                activeTab === tab.key
                  ? 'border-b-2 border-gold text-gold'
                  : 'text-text-tertiary hover:text-text-secondary',
              )}
              onClick={() => setActiveTab(tab.key)}
            >
              {tab.label}
              <span className="ml-1.5 text-[10px] opacity-40">
                {tab.shortcut}
              </span>
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
