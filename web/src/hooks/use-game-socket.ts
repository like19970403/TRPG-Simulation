import { useCallback, useEffect, useRef, useState } from 'react'
import { GameWebSocket } from '../api/websocket'
import { useGameStore } from '../stores/game-store'
import { useAuthStore } from '../stores/auth-store'
import { getSession } from '../api/sessions'
import { getScenario } from '../api/scenarios'
import { API } from '../lib/constants'
import type { ScenarioContent, ActionType, ActionPayloadMap } from '../api/types'

/** Type-safe sendAction signature. */
export type SendAction = <T extends ActionType>(type: T, payload: ActionPayloadMap[T]) => void

/** Try to refresh the access token via the refresh cookie. Returns the new token or null. */
async function refreshAccessToken(): Promise<string | null> {
  try {
    const res = await fetch(API.REFRESH, {
      method: 'POST',
      credentials: 'include',
    })
    if (!res.ok) return null
    const data = await res.json()
    useAuthStore.getState().setAuth(data.accessToken)
    return data.accessToken as string
  } catch {
    return null
  }
}

export function useGameSocket(sessionId: string) {
  const wsRef = useRef<GameWebSocket | null>(null)
  const [error, setError] = useState<string | null>(null)

  const connectionStatus = useGameStore((s) => s.connectionStatus)

  // Initialize: fetch session, fetch scenario, connect WebSocket
  useEffect(() => {
    let cancelled = false

    async function init() {
      try {
        useGameStore.getState().setConnectionStatus('connecting')

        // Fetch session info
        const session = await getSession(sessionId)
        if (cancelled) return
        useGameStore.getState().setSession(session)

        // Fetch scenario content for scene/item/NPC definitions
        try {
          const scenario = await getScenario(session.scenarioId)
          if (cancelled) return
          if (scenario.content) {
            useGameStore
              .getState()
              .setScenarioContent(
                scenario.content as unknown as ScenarioContent,
              )
          }
        } catch {
          // Scenario fetch failure is non-fatal; scene content may be unavailable
        }

        // Create WebSocket connection
        const ws = new GameWebSocket(
          sessionId,
          () => useAuthStore.getState().accessToken,
          () => useGameStore.getState().gameState?.last_sequence ?? 0,
          refreshAccessToken,
        )

        ws.onMessage = (envelope) => {
          useGameStore.getState().handleEvent(envelope)
        }

        ws.onOpen = () => {
          useGameStore.getState().setConnectionStatus('connected')
        }

        ws.onClose = () => {
          const store = useGameStore.getState()
          // Don't reconnect if the game has ended or we intentionally disconnected
          if (store.gameState?.status === 'completed') {
            store.setConnectionStatus('disconnected')
            ws.disconnect()
            return
          }
          if (store.connectionStatus !== 'disconnected') {
            store.setConnectionStatus('reconnecting')
          }
        }

        ws.onReconnectExhausted = () => {
          useGameStore.getState().setConnectionStatus('disconnected')
          setError('無法連線至遊戲伺服器，請重新整理頁面')
        }

        wsRef.current = ws
        ws.connect()
      } catch (err) {
        if (!cancelled) {
          useGameStore.getState().setConnectionStatus('disconnected')
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
      useGameStore.getState().clearGame()
    }
  }, [sessionId])

  const sendAction = useCallback(
    <T extends ActionType>(type: T, payload: ActionPayloadMap[T]) => {
      wsRef.current?.send({ type, payload })
    },
    [],
  )

  return { sendAction, connectionStatus, error }
}
