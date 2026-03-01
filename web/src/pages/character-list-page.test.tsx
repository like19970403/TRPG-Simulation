import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, cleanup } from '@testing-library/react'
import { CharacterListPage } from './character-list-page'
import type {
  CharacterListResponse,
  CharacterResponse,
} from '../api/types'

const mockListCharacters = vi.fn()
const mockDeleteCharacter = vi.fn()
vi.mock('../api/characters', () => ({
  listCharacters: (...args: unknown[]) => mockListCharacters(...args),
  deleteCharacter: (...args: unknown[]) => mockDeleteCharacter(...args),
  createCharacter: vi.fn(),
  updateCharacter: vi.fn(),
}))

function makeCharacter(
  overrides: Partial<CharacterResponse> = {},
): CharacterResponse {
  return {
    id: 'char-1',
    userId: 'u1',
    name: 'Aragorn',
    attributes: { STR: 16 },
    inventory: ['sword'],
    notes: 'Ranger',
    createdAt: '2026-03-01T00:00:00Z',
    updatedAt: '2026-03-01T00:00:00Z',
    ...overrides,
  }
}

function makeListResponse(
  characters: CharacterResponse[] = [],
  total = characters.length,
): CharacterListResponse {
  return { characters, total, limit: 20, offset: 0 }
}

describe('CharacterListPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  it('renders character cards after fetch', async () => {
    mockListCharacters.mockResolvedValueOnce(
      makeListResponse([
        makeCharacter({ id: '1', name: 'Aragorn' }),
        makeCharacter({ id: '2', name: 'Gandalf' }),
      ]),
    )

    render(<CharacterListPage />)

    await waitFor(() => {
      expect(screen.getByText('Aragorn')).toBeInTheDocument()
      expect(screen.getByText('Gandalf')).toBeInTheDocument()
    })
  })

  it('shows empty state when no characters', async () => {
    mockListCharacters.mockResolvedValueOnce(makeListResponse())

    render(<CharacterListPage />)

    await waitFor(() => {
      expect(
        screen.getByText('No characters yet. Create your first one!'),
      ).toBeInTheDocument()
    })
  })
})
