import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, cleanup } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { SessionListPage } from './session-list-page'
import type { SessionListResponse, SessionResponse } from '../api/types'

const mockListSessions = vi.fn()
vi.mock('../api/sessions', () => ({
  listSessions: (...args: unknown[]) => mockListSessions(...args),
}))

vi.mock('../api/scenarios', () => ({
  getScenario: () =>
    Promise.resolve({ id: 'sc-1', title: 'Test Scenario', status: 'published', content: {} }),
}))

vi.mock('react-router', async () => {
  const actual = await vi.importActual('react-router')
  return {
    ...actual,
    useNavigate: () => vi.fn(),
  }
})

function makeSession(overrides: Partial<SessionResponse> = {}): SessionResponse {
  return {
    id: 'session-1',
    scenarioId: 'sc-1',
    gmId: 'gm-1',
    status: 'lobby',
    inviteCode: 'ABC123',
    createdAt: '2026-03-01T12:00:00Z',
    startedAt: null,
    endedAt: null,
    ...overrides,
  }
}

function makeListResponse(
  sessions: SessionResponse[] = [],
  total = sessions.length,
): SessionListResponse {
  return { sessions, total, limit: 20, offset: 0 }
}

function renderPage() {
  return render(
    <MemoryRouter>
      <SessionListPage />
    </MemoryRouter>,
  )
}

describe('SessionListPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  it('renders session cards after fetch', async () => {
    mockListSessions.mockResolvedValueOnce(
      makeListResponse([
        makeSession({ id: '1', inviteCode: 'AAA111' }),
        makeSession({ id: '2', inviteCode: 'BBB222', status: 'active' }),
      ]),
    )
    renderPage()

    await waitFor(() => {
      expect(screen.getByText('AAA111')).toBeInTheDocument()
      expect(screen.getByText('BBB222')).toBeInTheDocument()
    })
  })

  it('shows empty state when no sessions', async () => {
    mockListSessions.mockResolvedValueOnce(makeListResponse())
    renderPage()

    await waitFor(() => {
      expect(screen.getByText('No sessions yet')).toBeInTheDocument()
    })
  })
})
