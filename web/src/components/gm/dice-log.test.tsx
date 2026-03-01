import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { DiceLog } from './dice-log'
import { useGameStore } from '../../stores/game-store'

beforeEach(() => {
  useGameStore.getState().clearGame()
})

afterEach(() => {
  cleanup()
})

function setupStoreWithDice() {
  useGameStore.getState().handleEvent({
    type: 'state_sync',
    session_id: 'session-1',
    sender_id: '',
    payload: {
      session_id: 'session-1',
      status: 'active',
      current_scene: 'scene-1',
      players: {},
      dice_history: [
        { formula: '2d6+3', results: [4, 5], modifier: 3, total: 12 },
        { formula: 'd20', results: [17], modifier: 0, total: 17 },
      ],
      variables: {},
      revealed_items: {},
      revealed_npc_fields: {},
      last_sequence: 1,
    },
    timestamp: Date.now(),
  })
}

describe('DiceLog', () => {
  it('renders dice history entries', () => {
    setupStoreWithDice()
    render(<DiceLog sendAction={vi.fn()} />)

    expect(screen.getByText('2d6+3')).toBeInTheDocument()
    expect(screen.getByText('= 12')).toBeInTheDocument()
    expect(screen.getByText('d20')).toBeInTheDocument()
    expect(screen.getByText('= 17')).toBeInTheDocument()
  })

  it('validates formula before sending', async () => {
    setupStoreWithDice()
    const sendAction = vi.fn()
    const user = userEvent.setup()

    render(<DiceLog sendAction={sendAction} />)

    // Type an invalid formula
    await user.type(screen.getByPlaceholderText('2d6+3'), 'invalid')
    await user.click(screen.getByRole('button', { name: 'Roll' }))

    expect(screen.getByText('Invalid formula (e.g. 2d6, d20+5)')).toBeInTheDocument()
    expect(sendAction).not.toHaveBeenCalled()
  })

  it('sends dice_roll action on valid input', async () => {
    setupStoreWithDice()
    const sendAction = vi.fn()
    const user = userEvent.setup()

    render(<DiceLog sendAction={sendAction} />)

    await user.type(screen.getByPlaceholderText('2d6+3'), '3d8+2')
    await user.click(screen.getByRole('button', { name: 'Roll' }))

    expect(sendAction).toHaveBeenCalledWith('dice_roll', {
      formula: '3d8+2',
    })
  })
})
