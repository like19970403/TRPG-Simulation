import { useCallback, useEffect, useRef, useState } from 'react'
import { GameWebSocket } from '../api/websocket'
import { useGameStore } from '../stores/game-store'
import { useAuthStore } from '../stores/auth-store'
import { getSession } from '../api/sessions'
import { getScenario } from '../api/scenarios'
import type { ScenarioContent } from '../api/types'

export function useGameSocket(sessionId: string) {
  const wsRef = useRef<GameWebSocket | null>(null)
  const [error, setError] = useState<string | null>(null)

  const {
    connectionStatus,
    setSession,
    setScenarioContent,
    setConnectionStatus,
    clearGame,
  } = useGameStore()

  // Initialize: fetch session, fetch scenario, connect WebSocket
  useEffect(() => {
    let cancelled = false

    async function init() {
      try {
        setConnectionStatus('connecting')

        // Fetch session info
        const session = await getSession(sessionId)
        if (cancelled) return
        setSession(session)

        // Fetch scenario content for scene/item/NPC definitions
        try {
          const scenario = await getScenario(session.scenarioId)
          if (cancelled) return
          if (scenario.content) {
            setScenarioContent(scenario.content as unknown as ScenarioContent)
          }
        } catch {
          // Scenario fetch failure is non-fatal; scene content may be unavailable
        }

        // Create WebSocket connection
        const ws = new GameWebSocket(
          sessionId,
          () => useAuthStore.getState().accessToken,
          () => useGameStore.getState().gameState?.last_sequence ?? 0,
        )

        ws.onMessage = (envelope) => {
          useGameStore.getState().handleEvent(envelope)
        }

        ws.onOpen = () => {
          useGameStore.getState().setConnectionStatus('connected')
        }

        ws.onClose = () => {
          const store = useGameStore.getState()
          if (store.connectionStatus !== 'disconnected') {
            store.setConnectionStatus('reconnecting')
          }
        }

        wsRef.current = ws
        ws.connect()
      } catch (err) {
        if (!cancelled) {
          setConnectionStatus('disconnected')
          setError(
            err instanceof Error ? err.message : 'Failed to initialize game',
          )
        }
      }
    }

    init()

    return () => {
      cancelled = true
      if (wsRef.current) {
        wsRef.current.disconnect()
        wsRef.current = null
      }
      clearGame()
    }
  }, [sessionId]) // eslint-disable-line react-hooks/exhaustive-deps

  const sendAction = useCallback(
    (type: string, payload: unknown) => {
      wsRef.current?.send({ type, payload })
    },
    [],
  )

  return { sendAction, connectionStatus, error }
}
