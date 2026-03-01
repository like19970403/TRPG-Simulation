import { create } from 'zustand'
import type {
  SessionResponse,
  ScenarioContent,
  GameState,
  WsEnvelope,
  EventLogEntry,
  ConnectionStatus,
  DiceResult,
} from '../api/types'

const MAX_EVENT_LOG = 200

interface GameStoreState {
  session: SessionResponse | null
  scenarioContent: ScenarioContent | null
  gameState: GameState | null
  eventLog: EventLogEntry[]
  connectionStatus: ConnectionStatus

  setSession: (session: SessionResponse) => void
  setScenarioContent: (content: ScenarioContent) => void
  handleEvent: (envelope: WsEnvelope) => void
  setConnectionStatus: (status: ConnectionStatus) => void
  clearGame: () => void
}

let eventSeqCounter = 0

export const useGameStore = create<GameStoreState>((set, get) => ({
  session: null,
  scenarioContent: null,
  gameState: null,
  eventLog: [],
  connectionStatus: 'disconnected',

  setSession: (session) => set({ session }),

  setScenarioContent: (content) => set({ scenarioContent: content }),

  handleEvent: (envelope) => {
    const { gameState } = get()
    const payload = envelope.payload as Record<string, unknown>

    // Build event log entry
    const logEntry: EventLogEntry = {
      id: `evt-${++eventSeqCounter}`,
      type: envelope.type,
      senderId: envelope.sender_id,
      payload: envelope.payload,
      timestamp: envelope.timestamp,
      sequence: eventSeqCounter,
    }

    let nextState = gameState

    switch (envelope.type) {
      case 'state_sync': {
        const raw = payload as unknown as Partial<GameState>
        nextState = {
          session_id: raw.session_id ?? '',
          status: raw.status ?? 'active',
          current_scene: raw.current_scene ?? '',
          players: raw.players ?? {},
          dice_history: raw.dice_history ?? [],
          variables: raw.variables ?? {},
          revealed_items: raw.revealed_items ?? {},
          revealed_npc_fields: raw.revealed_npc_fields ?? {},
          last_sequence: raw.last_sequence ?? 0,
        }
        break
      }
      case 'scene_changed': {
        if (nextState) {
          const scenePayload = payload as { scene_id: string }
          nextState = {
            ...nextState,
            current_scene: scenePayload.scene_id,
          }
        }
        break
      }
      case 'dice_rolled': {
        if (nextState) {
          const diceResult = payload as unknown as DiceResult
          nextState = {
            ...nextState,
            dice_history: [...nextState.dice_history, diceResult],
          }
        }
        break
      }
      case 'item_revealed': {
        if (nextState) {
          const itemPayload = payload as {
            item_id: string
            player_ids: string[]
          }
          const updated = { ...nextState.revealed_items }
          for (const pid of itemPayload.player_ids) {
            const existing = updated[pid] ?? []
            if (!existing.includes(itemPayload.item_id)) {
              updated[pid] = [...existing, itemPayload.item_id]
            }
          }
          nextState = { ...nextState, revealed_items: updated }
        }
        break
      }
      case 'npc_field_revealed': {
        if (nextState) {
          const npcPayload = payload as {
            npc_id: string
            field_key: string
            player_ids: string[]
          }
          const updated = { ...nextState.revealed_npc_fields }
          for (const pid of npcPayload.player_ids) {
            const npcMap = { ...(updated[pid] ?? {}) }
            const existing = npcMap[npcPayload.npc_id] ?? []
            if (!existing.includes(npcPayload.field_key)) {
              npcMap[npcPayload.npc_id] = [...existing, npcPayload.field_key]
            }
            updated[pid] = npcMap
          }
          nextState = { ...nextState, revealed_npc_fields: updated }
        }
        break
      }
      case 'variable_changed': {
        if (nextState) {
          const varPayload = payload as { name: string; new_value: unknown }
          nextState = {
            ...nextState,
            variables: {
              ...nextState.variables,
              [varPayload.name]: varPayload.new_value,
            },
          }
        }
        break
      }
      case 'game_paused': {
        if (nextState) {
          nextState = { ...nextState, status: 'paused' }
        }
        break
      }
      case 'game_resumed': {
        if (nextState) {
          nextState = { ...nextState, status: 'active' }
        }
        break
      }
      case 'game_ended': {
        if (nextState) {
          nextState = { ...nextState, status: 'completed' }
        }
        break
      }
      case 'player_joined': {
        if (nextState) {
          const joinPayload = payload as {
            user_id: string
            username: string
          }
          nextState = {
            ...nextState,
            players: {
              ...nextState.players,
              [joinPayload.user_id]: {
                user_id: joinPayload.user_id,
                username: joinPayload.username,
                current_scene: nextState.current_scene,
                online: true,
              },
            },
          }
        }
        break
      }
      case 'player_left': {
        if (nextState) {
          const leftPayload = payload as { user_id: string }
          const existing = nextState.players[leftPayload.user_id]
          if (existing) {
            nextState = {
              ...nextState,
              players: {
                ...nextState.players,
                [leftPayload.user_id]: { ...existing, online: false },
              },
            }
          }
        }
        break
      }
      // player_choice, gm_broadcast, error
      // → no state mutation, only log
    }

    set((s) => ({
      gameState: nextState,
      eventLog: [...s.eventLog, logEntry].slice(-MAX_EVENT_LOG),
    }))
  },

  setConnectionStatus: (status) => set({ connectionStatus: status }),

  clearGame: () => {
    eventSeqCounter = 0
    set({
      session: null,
      scenarioContent: null,
      gameState: null,
      eventLog: [],
      connectionStatus: 'disconnected',
    })
  },
}))
