import { describe, it, expect, beforeEach } from 'vitest'
import { useGameStore } from './game-store'
import type { WsEnvelope } from '../api/types'

function makeEnvelope(
  type: string,
  payload: unknown,
): WsEnvelope {
  return {
    type,
    session_id: 'session-1',
    sender_id: 'sender-1',
    payload,
    timestamp: Date.now(),
    // Sequence is embedded in the envelope for eventLog tracking
    // The store uses an internal counter
  }
}

describe('useGameStore', () => {
  beforeEach(() => {
    useGameStore.getState().clearGame()
  })

  it('starts with null gameState and empty eventLog', () => {
    const state = useGameStore.getState()
    expect(state.gameState).toBeNull()
    expect(state.scenarioContent).toBeNull()
    expect(state.session).toBeNull()
    expect(state.eventLog).toEqual([])
    expect(state.connectionStatus).toBe('disconnected')
  })

  it('handleEvent state_sync sets full gameState', () => {
    const gameState = {
      session_id: 'session-1',
      status: 'active',
      current_scene: 'scene-1',
      players: { 'p1': { user_id: 'u1', current_scene: 'scene-1' } },
      dice_history: [],
      variables: {},
      revealed_items: {},
      revealed_npc_fields: {},
      last_sequence: 5,
    }

    useGameStore.getState().handleEvent(
      makeEnvelope('state_sync', gameState),
    )

    const state = useGameStore.getState()
    expect(state.gameState).toEqual(gameState)
  })

  it('handleEvent scene_changed updates current_scene', () => {
    // First set up initial state via state_sync
    const gameState = {
      session_id: 'session-1',
      status: 'active',
      current_scene: 'scene-1',
      players: {},
      dice_history: [],
      variables: {},
      revealed_items: {},
      revealed_npc_fields: {},
      last_sequence: 1,
    }
    useGameStore.getState().handleEvent(
      makeEnvelope('state_sync', gameState),
    )

    // Now scene_changed
    useGameStore.getState().handleEvent(
      makeEnvelope('scene_changed', { scene_id: 'scene-2' }),
    )

    expect(useGameStore.getState().gameState?.current_scene).toBe('scene-2')
  })

  it('handleEvent dice_rolled appends to dice_history', () => {
    // Set up initial state
    const gameState = {
      session_id: 'session-1',
      status: 'active',
      current_scene: 'scene-1',
      players: {},
      dice_history: [],
      variables: {},
      revealed_items: {},
      revealed_npc_fields: {},
      last_sequence: 1,
    }
    useGameStore.getState().handleEvent(
      makeEnvelope('state_sync', gameState),
    )

    const diceResult = {
      formula: '2d6+3',
      results: [4, 5],
      modifier: 3,
      total: 12,
    }
    useGameStore.getState().handleEvent(
      makeEnvelope('dice_rolled', diceResult),
    )

    expect(useGameStore.getState().gameState?.dice_history).toHaveLength(1)
    expect(useGameStore.getState().gameState?.dice_history[0]).toEqual(diceResult)
  })

  it('event log caps at 200 entries (oldest dropped)', () => {
    // Set up initial state
    const gameState = {
      session_id: 'session-1',
      status: 'active',
      current_scene: 'scene-1',
      players: {},
      dice_history: [],
      variables: {},
      revealed_items: {},
      revealed_npc_fields: {},
      last_sequence: 0,
    }
    useGameStore.getState().handleEvent(
      makeEnvelope('state_sync', gameState),
    )

    // Push 201 events (state_sync above is event #1, then 200 more)
    for (let i = 0; i < 200; i++) {
      useGameStore.getState().handleEvent(
        makeEnvelope('variable_changed', {
          name: `var_${i}`,
          new_value: i,
        }),
      )
    }

    // state_sync + 200 variable_changed = 201 total, but capped at 200
    expect(useGameStore.getState().eventLog).toHaveLength(200)
    // The first event (state_sync) should have been dropped
    expect(useGameStore.getState().eventLog[0].type).toBe('variable_changed')
  })
})
