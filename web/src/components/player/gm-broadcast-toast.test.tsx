import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, cleanup, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { GmBroadcastToast } from './gm-broadcast-toast'
import { useGameStore } from '../../stores/game-store'

beforeEach(() => {
  useGameStore.getState().clearGame()
  vi.useFakeTimers({ shouldAdvanceTime: true })
})

afterEach(() => {
  vi.useRealTimers()
  cleanup()
})

function setupStoreAndBroadcast() {
  // Set up initial game state
  useGameStore.getState().handleEvent({
    type: 'state_sync',
    session_id: 'session-1',
    sender_id: '',
    payload: {
      session_id: 'session-1',
      status: 'active',
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

describe('GmBroadcastToast', () => {
  it('shows toast when gm_broadcast event arrives', () => {
    setupStoreAndBroadcast()
    render(<GmBroadcastToast />)

    // Push a gm_broadcast event
    act(() => {
      useGameStore.getState().handleEvent({
        type: 'gm_broadcast',
        session_id: 'session-1',
        sender_id: 'gm-1',
        payload: { content: 'Watch out for the dragon!' },
        timestamp: Date.now(),
      })
    })

    expect(screen.getByText('Watch out for the dragon!')).toBeInTheDocument()
  })

  it('hides toast after dismiss click', async () => {
    setupStoreAndBroadcast()
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
    render(<GmBroadcastToast />)

    // Push a gm_broadcast event
    act(() => {
      useGameStore.getState().handleEvent({
        type: 'gm_broadcast',
        session_id: 'session-1',
        sender_id: 'gm-1',
        payload: { content: 'A mysterious sound...' },
        timestamp: Date.now(),
      })
    })

    expect(screen.getByText('A mysterious sound...')).toBeInTheDocument()

    // Click dismiss
    await user.click(screen.getByRole('button', { name: '×' }))

    expect(screen.queryByText('A mysterious sound...')).not.toBeInTheDocument()
  })
})
