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
    // Set up game state with players
    useGameStore.getState().handleEvent({
      type: 'state_sync',
      session_id: 'session-1',
      sender_id: '',
      payload: {
        session_id: 'session-1',
        status: 'active',
        current_scene: 'scene-1',
        players: {
          Luna: { user_id: 'u1', current_scene: 'scene-1' },
          Kai: { user_id: 'u2', current_scene: 'scene-1' },
        },
        dice_history: [],
        variables: {},
        revealed_items: {},
        revealed_npc_fields: {},
        last_sequence: 1,
      },
      timestamp: Date.now(),
    })

    render(<PlayerPanel />)

    expect(screen.getByText('Luna')).toBeInTheDocument()
    expect(screen.getByText('Kai')).toBeInTheDocument()
  })

  it('shows correct player count', () => {
    useGameStore.getState().handleEvent({
      type: 'state_sync',
      session_id: 'session-1',
      sender_id: '',
      payload: {
        session_id: 'session-1',
        status: 'active',
        current_scene: 'scene-1',
        players: {
          Luna: { user_id: 'u1', current_scene: 'scene-1' },
          Kai: { user_id: 'u2', current_scene: 'scene-1' },
          Frey: { user_id: 'u3', current_scene: 'scene-1' },
        },
        dice_history: [],
        variables: {},
        revealed_items: {},
        revealed_npc_fields: {},
        last_sequence: 1,
      },
      timestamp: Date.now(),
    })

    render(<PlayerPanel />)

    expect(screen.getByText('3')).toBeInTheDocument()
  })
})
