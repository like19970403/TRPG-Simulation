import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { SceneView } from './scene-view'
import { useGameStore } from '../../stores/game-store'
import type { ScenarioContent } from '../../api/types'

const mockScenario: ScenarioContent = {
  id: 'scenario-1',
  title: 'Test Scenario',
  start_scene: 'scene-1',
  scenes: [
    {
      id: 'scene-1',
      name: 'Dark Chamber',
      content: 'You stand in a dim chamber. Torches flicker on the walls.',
      gm_notes: 'Secret: The east wall has a hidden door.',
      transitions: [
        { target: 'scene-2', trigger: 'player_choice', label: 'Open the door' },
        { target: 'scene-3', trigger: 'auto' },
        { target: 'scene-4', trigger: 'condition_met', label: 'Sneak past guard' },
        { target: 'scene-5', trigger: 'player_choice', label: 'Search the room' },
      ],
    },
  ],
  items: [],
  npcs: [],
  variables: [],
}

beforeEach(() => {
  useGameStore.getState().clearGame()
})

afterEach(() => {
  cleanup()
})

function setupStore() {
  useGameStore.getState().setScenarioContent(mockScenario)
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

describe('SceneView', () => {
  it('renders scene name and content', () => {
    setupStore()
    render(<SceneView sendAction={vi.fn()} />)

    expect(screen.getByText('Dark Chamber')).toBeInTheDocument()
    expect(
      screen.getByText('You stand in a dim chamber. Torches flicker on the walls.'),
    ).toBeInTheDocument()
  })

  it('renders player_choice transitions as vote buttons and calls sendAction', async () => {
    setupStore()
    const sendAction = vi.fn()
    const user = userEvent.setup()

    render(<SceneView sendAction={sendAction} />)

    // player_choice transitions should be visible
    const openDoor = screen.getByText('Open the door')
    const searchRoom = screen.getByText('Search the room')
    expect(openDoor).toBeInTheDocument()
    expect(searchRoom).toBeInTheDocument()

    // auto and condition_met transitions should NOT be visible
    expect(screen.queryByText('Sneak past guard')).not.toBeInTheDocument()

    // Click a vote button
    await user.click(openDoor)
    expect(sendAction).toHaveBeenCalledWith('player_choice', {
      transition_index: 0,
    })
    // Store should track the vote
    expect(useGameStore.getState().myVoteIndex).toBe(0)
  })

  it('shows vote count badge when votes exist', () => {
    setupStore()
    // Simulate votes arriving from server
    useGameStore.getState().handleEvent({
      type: 'player_votes',
      session_id: 'session-1',
      sender_id: '',
      payload: {
        votes: { '0': { count: 2, voters: ['Alice', 'Bob'] } },
      },
      timestamp: Date.now(),
    })

    render(<SceneView sendAction={vi.fn()} />)

    expect(screen.getByText('2 票')).toBeInTheDocument()
  })

  it('highlights voted button with ring', async () => {
    setupStore()
    const user = userEvent.setup()

    render(<SceneView sendAction={vi.fn()} />)

    const openDoor = screen.getByText('Open the door')
    await user.click(openDoor)

    // After voting, the "已投票" indicator should appear
    expect(screen.getByText('已投票')).toBeInTheDocument()
  })

  it('does NOT render gm_notes', () => {
    setupStore()
    render(<SceneView sendAction={vi.fn()} />)

    expect(
      screen.queryByText('Secret: The east wall has a hidden door.'),
    ).not.toBeInTheDocument()
    expect(screen.queryByText('GM Notes')).not.toBeInTheDocument()
  })
})
