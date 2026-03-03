import { create } from 'zustand'
import type {
  SessionResponse,
  ScenarioContent,
  GameState,
  WsEnvelope,
  EventLogEntry,
  ConnectionStatus,
  DiceResult,
  VoteTallyEntry,
} from '../api/types'

const MAX_EVENT_LOG = 200

/** Filtered transition sent by the server (includes original index) */
export interface FilteredTransition {
  target: string
  trigger: string
  label?: string
  transition_index: string
}

/** The active scene as sent by the server (per-client filtered) */
export interface ActiveScene {
  id: string
  name: string
  content: string
  transitions: FilteredTransition[]
}

interface GameStoreState {
  session: SessionResponse | null
  scenarioContent: ScenarioContent | null
  gameState: GameState | null
  activeScene: ActiveScene | null
  currentVotes: Record<string, VoteTallyEntry>
  myVoteIndex: number | null
  eventLog: EventLogEntry[]
  connectionStatus: ConnectionStatus

  setSession: (session: SessionResponse) => void
  setScenarioContent: (content: ScenarioContent) => void
  handleEvent: (envelope: WsEnvelope) => void
  setConnectionStatus: (status: ConnectionStatus) => void
  setMyVote: (transitionIndex: number) => void
  clearGame: () => void
}

let eventSeqCounter = 0

export const useGameStore = create<GameStoreState>((set, get) => ({
  session: null,
  scenarioContent: null,
  gameState: null,
  activeScene: null,
  currentVotes: {},
  myVoteIndex: null,
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
    let newActiveScene: ActiveScene | null | undefined
    let shouldResetVotes = false

    switch (envelope.type) {
      case 'state_sync': {
        const raw = payload as unknown as Partial<GameState>
        nextState = {
          session_id: raw.session_id ?? '',
          status: raw.status ?? 'active',
          current_scene: raw.current_scene ?? '',
          players: raw.players ?? {},
          player_attributes: raw.player_attributes ?? {},
          dice_history: raw.dice_history ?? [],
          variables: raw.variables ?? {},
          revealed_items: raw.revealed_items ?? {},
          revealed_npc_fields: raw.revealed_npc_fields ?? {},
          player_inventory: raw.player_inventory ?? {},
          last_sequence: raw.last_sequence ?? 0,
        }
        break
      }
      case 'scene_changed': {
        if (nextState) {
          const scenePayload = payload as {
            scene_id: string
            scene?: ActiveScene
          }
          nextState = {
            ...nextState,
            current_scene: scenePayload.scene_id,
          }
          if (scenePayload.scene) {
            newActiveScene = {
              id: scenePayload.scene_id,
              name: scenePayload.scene.name ?? '',
              content: scenePayload.scene.content ?? '',
              transitions: scenePayload.scene.transitions ?? [],
            }
          }
        }
        shouldResetVotes = true
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
          const updatedInv = { ...nextState.player_inventory }
          for (const pid of itemPayload.player_ids) {
            const existing = updated[pid] ?? []
            if (!existing.includes(itemPayload.item_id)) {
              updated[pid] = [...existing, itemPayload.item_id]
            }
            // Backward compat: also add to player_inventory (qty 1)
            const inv = [...(updatedInv[pid] ?? [])]
            const idx = inv.findIndex((e) => e.item_id === itemPayload.item_id)
            if (idx === -1) {
              inv.push({ item_id: itemPayload.item_id, quantity: 1 })
            }
            updatedInv[pid] = inv
          }
          nextState = { ...nextState, revealed_items: updated, player_inventory: updatedInv }
        }
        break
      }
      case 'item_given': {
        if (nextState) {
          const givePayload = payload as {
            item_id: string
            player_ids: string[]
            quantity?: number
          }
          const qty = givePayload.quantity ?? 1
          const updatedInv = { ...nextState.player_inventory }
          for (const pid of givePayload.player_ids) {
            const inv = [...(updatedInv[pid] ?? [])]
            const idx = inv.findIndex((e) => e.item_id === givePayload.item_id)
            if (idx >= 0) {
              inv[idx] = { ...inv[idx], quantity: inv[idx].quantity + qty }
            } else {
              inv.push({ item_id: givePayload.item_id, quantity: qty })
            }
            updatedInv[pid] = inv
          }
          nextState = { ...nextState, player_inventory: updatedInv }
        }
        break
      }
      case 'item_removed': {
        if (nextState) {
          const removePayload = payload as {
            item_id: string
            player_ids: string[]
            quantity?: number
          }
          const qty = removePayload.quantity ?? 1
          const updatedInv = { ...nextState.player_inventory }
          for (const pid of removePayload.player_ids) {
            const inv = [...(updatedInv[pid] ?? [])]
            const idx = inv.findIndex((e) => e.item_id === removePayload.item_id)
            if (idx >= 0) {
              if (qty === 0 || inv[idx].quantity <= qty) {
                inv.splice(idx, 1)
              } else {
                inv[idx] = { ...inv[idx], quantity: inv[idx].quantity - qty }
              }
            }
            updatedInv[pid] = inv
          }
          nextState = { ...nextState, player_inventory: updatedInv }
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
      case 'transitions_updated': {
        const tuPayload = payload as {
          scene_id: string
          transitions: FilteredTransition[]
        }
        const { activeScene: curScene } = get()
        if (curScene && curScene.id === tuPayload.scene_id) {
          newActiveScene = {
            ...curScene,
            transitions: tuPayload.transitions ?? [],
          }
        }
        // Don't add to event log — silent UI refresh.
        set(
          newActiveScene !== undefined
            ? { activeScene: newActiveScene }
            : {},
        )
        return
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
            character_id?: string
            character_name?: string
            attributes?: Record<string, unknown>
          }
          const updatedAttrs = joinPayload.attributes
            ? {
                ...nextState.player_attributes,
                [joinPayload.user_id]: joinPayload.attributes,
              }
            : nextState.player_attributes
          nextState = {
            ...nextState,
            players: {
              ...nextState.players,
              [joinPayload.user_id]: {
                user_id: joinPayload.user_id,
                username: joinPayload.username,
                character_id: joinPayload.character_id,
                character_name: joinPayload.character_name,
                current_scene: nextState.current_scene,
                online: true,
              },
            },
            player_attributes: updatedAttrs,
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
      case 'player_votes': {
        const votesPayload = payload as {
          votes: Record<string, VoteTallyEntry>
        }
        set({ currentVotes: votesPayload.votes ?? {} })
        // Don't add to event log (too frequent). Return early.
        return
      }
      // player_choice, gm_broadcast, error
      // → no state mutation, only log
    }

    set((s) => ({
      gameState: nextState,
      ...(newActiveScene !== undefined ? { activeScene: newActiveScene } : {}),
      ...(shouldResetVotes ? { currentVotes: {}, myVoteIndex: null } : {}),
      eventLog: [...s.eventLog, logEntry].slice(-MAX_EVENT_LOG),
    }))
  },

  setConnectionStatus: (status) => set({ connectionStatus: status }),

  setMyVote: (transitionIndex) => set({ myVoteIndex: transitionIndex }),

  clearGame: () => {
    eventSeqCounter = 0
    set({
      session: null,
      scenarioContent: null,
      gameState: null,
      activeScene: null,
      currentVotes: {},
      myVoteIndex: null,
      eventLog: [],
      connectionStatus: 'disconnected',
    })
  },
}))
