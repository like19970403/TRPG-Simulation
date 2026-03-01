import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ScenePanel } from './scene-panel'
import { useGameStore } from '../../stores/game-store'
import type { ScenarioContent } from '../../api/types'

const mockScenario: ScenarioContent = {
  id: 'scenario-1',
  title: 'Test Scenario',
  start_scene: 'scene-1',
  scenes: [
    {
      id: 'scene-1',
      name: 'Main Hall',
      content: 'You enter a dark hall with torches on the walls.',
      gm_notes: 'Secret: There is a hidden passage behind the painting.',
      transitions: [
        { target: 'scene-2', trigger: 'player_choice', label: 'Enter Library' },
        { target: 'scene-3', trigger: 'auto' },
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

describe('ScenePanel', () => {
  it('renders scene name and content', () => {
    setupStore()
    render(<ScenePanel sendAction={vi.fn()} />)

    expect(screen.getByText('Main Hall')).toBeInTheDocument()
    expect(
      screen.getByText('You enter a dark hall with torches on the walls.'),
    ).toBeInTheDocument()
  })

  it('renders GM notes section', () => {
    setupStore()
    render(<ScenePanel sendAction={vi.fn()} />)

    expect(screen.getByText('GM Notes')).toBeInTheDocument()
    expect(
      screen.getByText(
        'Secret: There is a hidden passage behind the painting.',
      ),
    ).toBeInTheDocument()
  })

  it('renders transition buttons and calls sendAction on click', async () => {
    setupStore()
    const sendAction = vi.fn()
    const user = userEvent.setup()

    render(<ScenePanel sendAction={sendAction} />)

    const libraryBtn = screen.getByText('Enter Library')
    expect(libraryBtn).toBeInTheDocument()

    await user.click(libraryBtn)
    expect(sendAction).toHaveBeenCalledWith('advance_scene', {
      scene_id: 'scene-2',
    })
  })
})
