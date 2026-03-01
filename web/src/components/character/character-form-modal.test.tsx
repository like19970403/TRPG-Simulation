import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, cleanup } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { CharacterFormModal } from './character-form-modal'

const mockCreateCharacter = vi.fn()
const mockUpdateCharacter = vi.fn()
vi.mock('../../api/characters', () => ({
  createCharacter: (...args: unknown[]) => mockCreateCharacter(...args),
  updateCharacter: (...args: unknown[]) => mockUpdateCharacter(...args),
}))

describe('CharacterFormModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  it('creates a new character with valid data', async () => {
    mockCreateCharacter.mockResolvedValueOnce({
      id: 'char-1',
      userId: 'u1',
      name: 'Aragorn',
      attributes: { STR: 16 },
      inventory: ['sword'],
      notes: 'Ranger',
      createdAt: '2026-03-01T00:00:00Z',
      updatedAt: '2026-03-01T00:00:00Z',
    })

    const onSaved = vi.fn()
    const user = userEvent.setup()

    render(
      <CharacterFormModal open={true} onClose={vi.fn()} onSaved={onSaved} />,
    )

    await user.type(screen.getByLabelText('Name'), 'Aragorn')
    await user.type(screen.getByLabelText('Notes'), 'Ranger')

    // userEvent.type interprets { } as special keys; use paste instead
    const attrsInput = screen.getByLabelText('Attributes (JSON)')
    await user.clear(attrsInput)
    await user.click(attrsInput)
    await user.paste('{"STR": 16}')

    const invInput = screen.getByLabelText('Inventory (JSON)')
    await user.clear(invInput)
    await user.click(invInput)
    await user.paste('["sword"]')

    await user.click(screen.getByRole('button', { name: 'Create' }))

    await waitFor(() => {
      expect(mockCreateCharacter).toHaveBeenCalledWith({
        name: 'Aragorn',
        attributes: { STR: 16 },
        inventory: ['sword'],
        notes: 'Ranger',
      })
      expect(onSaved).toHaveBeenCalled()
    })
  })

  it('edits an existing character and calls update', async () => {
    mockUpdateCharacter.mockResolvedValueOnce({
      id: 'char-1',
      userId: 'u1',
      name: 'Aragorn II',
      attributes: { STR: 18 },
      inventory: [],
      notes: 'King',
      createdAt: '2026-03-01T00:00:00Z',
      updatedAt: '2026-03-01T01:00:00Z',
    })

    const onSaved = vi.fn()
    const user = userEvent.setup()

    render(
      <CharacterFormModal
        open={true}
        onClose={vi.fn()}
        onSaved={onSaved}
        character={{
          id: 'char-1',
          userId: 'u1',
          name: 'Aragorn',
          attributes: { STR: 16 },
          inventory: ['sword'],
          notes: 'Ranger',
          createdAt: '2026-03-01T00:00:00Z',
          updatedAt: '2026-03-01T00:00:00Z',
        }}
      />,
    )

    const nameInput = screen.getByLabelText('Name')
    await user.clear(nameInput)
    await user.type(nameInput, 'Aragorn II')

    await user.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => {
      expect(mockUpdateCharacter).toHaveBeenCalledWith('char-1', {
        name: 'Aragorn II',
        attributes: { STR: 16 },
        inventory: ['sword'],
        notes: 'Ranger',
      })
      expect(onSaved).toHaveBeenCalled()
    })
  })
})
