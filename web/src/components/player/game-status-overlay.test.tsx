import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { GameStatusOverlay } from './game-status-overlay'
import { useGameStore } from '../../stores/game-store'

beforeEach(() => {
  useGameStore.getState().clearGame()
})

afterEach(() => {
  cleanup()
})

function setupStore(status: string) {
  useGameStore.getState().handleEvent({
    type: 'state_sync',
    session_id: 'session-1',
    sender_id: '',
    payload: {
      session_id: 'session-1',
      status,
      current_scene: 'scene-1',
      players: {},
      dice_history: [],
      variables: {},
      revealed_items: {},
      revealed_npc_fields: {},
      last_sequence: 1,
    },
    timestamp: Date.now(),
  })
}

describe('GameStatusOverlay', () => {
  it('shows "Game Paused" overlay when status is paused', () => {
    setupStore('paused')

    render(
      <MemoryRouter>
        <GameStatusOverlay />
      </MemoryRouter>,
    )

    expect(screen.getByText('Game Paused')).toBeInTheDocument()
    expect(
      screen.getByText('Waiting for GM to resume...'),
    ).toBeInTheDocument()
  })

  it('shows "Game Over" overlay with return button when status is completed', () => {
    setupStore('completed')

    render(
      <MemoryRouter>
        <GameStatusOverlay />
      </MemoryRouter>,
    )

    expect(screen.getByText('Game Over')).toBeInTheDocument()
    expect(
      screen.getByRole('link', { name: 'Return to Dashboard' }),
    ).toBeInTheDocument()
  })
})
