import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import { PlayerPanel } from './player-panel'
import { useGameStore } from '../../stores/game-store'

beforeEach(() => {
  useGameStore.getState().clearGame()
})

afterEach(() => {
  cleanup()
})

describe('PlayerPanel', () => {
  it('renders player list from store', () => {
    useGameStore.getState().handleEvent({
      type: 'state_sync',
      session_id: 'session-1',
      sender_id: '',
      payload: {
        session_id: 'session-1',
        status: 'active',
        current_scene: 'scene-1',
        players: {
          u1: { user_id: 'u1', username: 'Luna', current_scene: 'scene-1', online: true },
          u2: { user_id: 'u2', username: 'Kai', current_scene: 'scene-1', online: true },
        },
        player_attributes: {},
        dice_history: [],
        variables: {},
        revealed_items: {},
        revealed_npc_fields: {},
        player_inventory: {},
        last_sequence: 1,
      },
      timestamp: Date.now(),
    })

    render(<PlayerPanel />)

    expect(screen.getByText('Luna')).toBeInTheDocument()
    expect(screen.getByText('Kai')).toBeInTheDocument()
  })

  it('shows correct online player count', () => {
    useGameStore.getState().handleEvent({
      type: 'state_sync',
      session_id: 'session-1',
      sender_id: '',
      payload: {
        session_id: 'session-1',
        status: 'active',
        current_scene: 'scene-1',
        players: {
          u1: { user_id: 'u1', username: 'Luna', current_scene: 'scene-1', online: true },
          u2: { user_id: 'u2', username: 'Kai', current_scene: 'scene-1', online: true },
          u3: { user_id: 'u3', username: 'Frey', current_scene: 'scene-1', online: true },
        },
        player_attributes: {},
        dice_history: [],
        variables: {},
        revealed_items: {},
        revealed_npc_fields: {},
        player_inventory: {},
        last_sequence: 1,
      },
      timestamp: Date.now(),
    })

    render(<PlayerPanel />)

    expect(screen.getByText('3')).toBeInTheDocument()
  })

  it('displays character name and attributes', () => {
    useGameStore.getState().setScenarioContent({
      id: 'sc-1',
      title: 'Test',
      start_scene: 'scene-1',
      scenes: [],
      items: [],
      npcs: [],
      variables: [],
      rules: {
        attributes: [
          { name: '勇氣', display: '勇氣', default: 0 },
          { name: '感知', display: '感知', default: 0 },
        ],
      },
    })
    useGameStore.getState().handleEvent({
      type: 'state_sync',
      session_id: 'session-1',
      sender_id: '',
      payload: {
        session_id: 'session-1',
        status: 'active',
        current_scene: 'scene-1',
        players: {
          u1: {
            user_id: 'u1',
            username: 'player1',
            character_id: 'ch-1',
            character_name: '勇者小明',
            current_scene: 'scene-1',
            online: true,
          },
        },
        player_attributes: {
          u1: { '勇氣': 5, '感知': 3 },
        },
        dice_history: [],
        variables: {},
        revealed_items: {},
        revealed_npc_fields: {},
        player_inventory: {},
        last_sequence: 1,
      },
      timestamp: Date.now(),
    })

    render(<PlayerPanel />)

    // Character name shown as primary
    expect(screen.getByText('勇者小明')).toBeInTheDocument()
    // Username shown as secondary
    expect(screen.getByText('player1')).toBeInTheDocument()
    // Attributes displayed
    expect(screen.getByText('5')).toBeInTheDocument()
    expect(screen.getByText('3')).toBeInTheDocument()
  })
})
