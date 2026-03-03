import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import { InventorySidebar } from './inventory-sidebar'
import { useGameStore } from '../../stores/game-store'
import type { ScenarioContent, InventoryEntry } from '../../api/types'

// Mock auth store — player user
vi.mock('../../stores/auth-store', () => ({
  useAuthStore: (selector: (s: { user: { id: string } }) => unknown) =>
    selector({ user: { id: 'player-1' } }),
}))

const mockScenario: ScenarioContent = {
  id: 'scenario-1',
  title: 'Test Scenario',
  start_scene: 'scene-1',
  scenes: [],
  items: [
    { id: 'item-1', name: 'Ancient Key', type: 'key_item', description: 'A rusty key' },
    { id: 'item-2', name: 'Old Map', type: 'clue', description: 'A faded map' },
    { id: 'item-3', name: 'Hidden Gem', type: 'treasure', description: 'A sparkling gem' },
  ],
  npcs: [],
  variables: [],
}

beforeEach(() => {
  useGameStore.getState().clearGame()
})

afterEach(() => {
  cleanup()
})

function setupStore(playerInventory: Record<string, InventoryEntry[]>) {
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
      player_attributes: {},
      dice_history: [],
      variables: {},
      revealed_items: {},
      revealed_npc_fields: {},
      player_inventory: playerInventory,
      last_sequence: 1,
    },
    timestamp: Date.now(),
  })
}

describe('InventorySidebar', () => {
  it('renders inventory items for current player', () => {
    setupStore({
      'player-1': [
        { item_id: 'item-1', quantity: 1 },
        { item_id: 'item-2', quantity: 1 },
      ],
    })

    render(<InventorySidebar onItemClick={vi.fn()} />)

    expect(screen.getByText('Ancient Key')).toBeInTheDocument()
    expect(screen.getByText('Old Map')).toBeInTheDocument()
    // item-3 is NOT in this player's inventory
    expect(screen.queryByText('Hidden Gem')).not.toBeInTheDocument()
  })

  it('shows empty message when inventory is empty', () => {
    setupStore({})

    render(<InventorySidebar onItemClick={vi.fn()} />)

    expect(screen.getByText('背包是空的')).toBeInTheDocument()
  })

  it('shows quantity badge for stackable items', () => {
    setupStore({
      'player-1': [
        { item_id: 'item-1', quantity: 3 },
      ],
    })

    render(<InventorySidebar onItemClick={vi.fn()} />)

    expect(screen.getByText('Ancient Key')).toBeInTheDocument()
    expect(screen.getByText('x3')).toBeInTheDocument()
  })
})
