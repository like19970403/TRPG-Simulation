import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { render, screen, waitFor, cleanup } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Routes, Route } from 'react-router'
import { ScenarioEditPage } from './scenario-edit-page'
import type { ScenarioResponse } from '../api/types'

const mockCreateScenario = vi.fn()
const mockUpdateScenario = vi.fn()
const mockGetScenario = vi.fn()
vi.mock('../api/scenarios', () => ({
  createScenario: (...args: unknown[]) => mockCreateScenario(...args),
  updateScenario: (...args: unknown[]) => mockUpdateScenario(...args),
  getScenario: (...args: unknown[]) => mockGetScenario(...args),
}))

const mockNavigate = vi.fn()
vi.mock('react-router', async () => {
  const actual = await vi.importActual('react-router')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

function makeScenario(overrides: Partial<ScenarioResponse> = {}): ScenarioResponse {
  return {
    id: 'abc-123',
    authorId: 'author-1',
    title: 'Existing Scenario',
    description: 'An existing adventure',
    version: 1,
    status: 'draft',
    content: { startScene: 'entrance', scenes: [] },
    createdAt: '2026-02-28T12:00:00Z',
    updatedAt: '2026-02-28T12:00:00Z',
    ...overrides,
  }
}

function renderCreateMode() {
  return render(
    <MemoryRouter initialEntries={['/scenarios/new']}>
      <Routes>
        <Route path="/scenarios/new" element={<ScenarioEditPage />} />
      </Routes>
    </MemoryRouter>,
  )
}

function renderEditMode() {
  return render(
    <MemoryRouter initialEntries={['/scenarios/abc-123/edit']}>
      <Routes>
        <Route path="/scenarios/:id/edit" element={<ScenarioEditPage />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('ScenarioEditPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  it('renders create form with empty fields', () => {
    renderCreateMode()

    expect(screen.getByText('New Scenario')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Enter scenario title')).toHaveValue('')
    expect(
      screen.getByPlaceholderText('Brief description of the scenario'),
    ).toHaveValue('')
  })

  it('renders edit form with pre-filled data', async () => {
    mockGetScenario.mockResolvedValueOnce(makeScenario())
    renderEditMode()

    await waitFor(() => {
      expect(screen.getByText('Edit Scenario')).toBeInTheDocument()
    })
    expect(screen.getByPlaceholderText('Enter scenario title')).toHaveValue(
      'Existing Scenario',
    )
    expect(
      screen.getByPlaceholderText('Brief description of the scenario'),
    ).toHaveValue('An existing adventure')
  })

  it('shows validation error for empty title', async () => {
    renderCreateMode()
    const user = userEvent.setup()

    await user.click(screen.getByText('Save Draft'))

    expect(screen.getByText('Title is required')).toBeInTheDocument()
    expect(mockCreateScenario).not.toHaveBeenCalled()
  })

  it('shows validation error for invalid JSON', async () => {
    renderCreateMode()
    const user = userEvent.setup()

    await user.type(
      screen.getByPlaceholderText('Enter scenario title'),
      'Test',
    )
    // userEvent.type interprets { as special key, use {{} to type literal {
    await user.type(
      screen.getByPlaceholderText('{"startScene": "entrance", "scenes": [...]}'),
      '{{}invalid json',
    )
    await user.click(screen.getByText('Save Draft'))

    expect(screen.getByText('Invalid JSON syntax')).toBeInTheDocument()
    expect(mockCreateScenario).not.toHaveBeenCalled()
  })

  it('calls createScenario and navigates on success', async () => {
    mockCreateScenario.mockResolvedValueOnce(makeScenario({ id: 'new-id' }))
    renderCreateMode()
    const user = userEvent.setup()

    await user.type(
      screen.getByPlaceholderText('Enter scenario title'),
      'My Adventure',
    )
    await user.click(screen.getByText('Save Draft'))

    await waitFor(() => {
      expect(mockCreateScenario).toHaveBeenCalledWith({
        title: 'My Adventure',
        description: '',
        content: {},
      })
    })
    expect(mockNavigate).toHaveBeenCalledWith('/scenarios/new-id')
  })

  it('calls updateScenario and navigates on success', async () => {
    mockGetScenario.mockResolvedValueOnce(makeScenario())
    mockUpdateScenario.mockResolvedValueOnce(makeScenario())
    renderEditMode()

    await waitFor(() => {
      expect(screen.getByPlaceholderText('Enter scenario title')).toHaveValue(
        'Existing Scenario',
      )
    })

    const user = userEvent.setup()
    const titleInput = screen.getByPlaceholderText('Enter scenario title')
    await user.clear(titleInput)
    await user.type(titleInput, 'Updated Title')
    await user.click(screen.getByText('Save Draft'))

    await waitFor(() => {
      expect(mockUpdateScenario).toHaveBeenCalledWith('abc-123', {
        title: 'Updated Title',
        description: 'An existing adventure',
        content: { startScene: 'entrance', scenes: [] },
      })
    })
    expect(mockNavigate).toHaveBeenCalledWith('/scenarios/abc-123')
  })

  it('shows server error on API failure', async () => {
    const { ApiClientError } = await import('../api/client')
    mockCreateScenario.mockRejectedValueOnce(
      new ApiClientError(400, {
        error: 'VALIDATION_ERROR',
        message: 'Title is too long',
      }),
    )
    renderCreateMode()
    const user = userEvent.setup()

    await user.type(
      screen.getByPlaceholderText('Enter scenario title'),
      'Test',
    )
    await user.click(screen.getByText('Save Draft'))

    await waitFor(() => {
      expect(screen.getByText('Title is too long')).toBeInTheDocument()
    })
  })
})
