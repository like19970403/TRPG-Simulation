import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { render, screen, waitFor, cleanup } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { ScenarioListPage } from './scenario-list-page'
import type { ScenarioListResponse, ScenarioResponse } from '../api/types'

const mockListScenarios = vi.fn()
vi.mock('../api/scenarios', () => ({
  listScenarios: (...args: unknown[]) => mockListScenarios(...args),
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
    id: 'test-id-1',
    authorId: 'author-1',
    title: 'Haunted Mansion',
    description: 'A spooky adventure',
    version: 1,
    status: 'draft',
    content: { scenes: [] },
    createdAt: '2026-02-28T12:00:00Z',
    updatedAt: '2026-02-28T12:00:00Z',
    ...overrides,
  }
}

function makeListResponse(
  scenarios: ScenarioResponse[] = [],
  total = scenarios.length,
): ScenarioListResponse {
  return { scenarios, total, limit: 20, offset: 0 }
}

function renderPage() {
  return render(
    <MemoryRouter>
      <ScenarioListPage />
    </MemoryRouter>,
  )
}

describe('ScenarioListPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  it('renders heading and New Scenario button', async () => {
    mockListScenarios.mockResolvedValueOnce(makeListResponse())
    renderPage()

    expect(screen.getByText('Scenarios')).toBeInTheDocument()
    expect(screen.getByText('+ New Scenario')).toBeInTheDocument()
    await waitFor(() => expect(mockListScenarios).toHaveBeenCalled())
  })

  it('shows loading spinner initially', () => {
    mockListScenarios.mockReturnValueOnce(new Promise(() => {}))
    renderPage()

    expect(document.querySelector('.animate-spin')).toBeInTheDocument()
  })

  it('renders scenario cards after fetch', async () => {
    mockListScenarios.mockResolvedValueOnce(
      makeListResponse([
        makeScenario({ id: '1', title: 'Adventure One', status: 'draft' }),
        makeScenario({ id: '2', title: 'Adventure Two', status: 'published' }),
      ]),
    )
    renderPage()

    await waitFor(() => {
      expect(screen.getByText('Adventure One')).toBeInTheDocument()
      expect(screen.getByText('Adventure Two')).toBeInTheDocument()
    })
  })

  it('shows empty state when no scenarios', async () => {
    mockListScenarios.mockResolvedValueOnce(makeListResponse())
    renderPage()

    await waitFor(() => {
      expect(
        screen.getByText('No scenarios yet. Create your first one!'),
      ).toBeInTheDocument()
    })
  })

  it('filters scenarios by tab', async () => {
    mockListScenarios.mockResolvedValueOnce(
      makeListResponse([
        makeScenario({ id: '1', title: 'Draft One', status: 'draft' }),
        makeScenario({ id: '2', title: 'Published One', status: 'published' }),
      ]),
    )
    renderPage()
    const user = userEvent.setup()

    await waitFor(() => {
      expect(screen.getByText('Draft One')).toBeInTheDocument()
      expect(screen.getByText('Published One')).toBeInTheDocument()
    })

    // Click the "Draft" tab button (not the badge)
    const draftTab = screen.getByRole('button', { name: 'Draft' })
    await user.click(draftTab)

    expect(screen.getByText('Draft One')).toBeInTheDocument()
    expect(screen.queryByText('Published One')).not.toBeInTheDocument()
  })

  it('shows error message on API failure', async () => {
    const { ApiClientError } = await import('../api/client')
    mockListScenarios.mockRejectedValueOnce(
      new ApiClientError(500, {
        error: 'INTERNAL_ERROR',
        message: 'Something went wrong',
      }),
    )
    renderPage()

    await waitFor(() => {
      expect(screen.getByText('Something went wrong')).toBeInTheDocument()
    })
  })
})
