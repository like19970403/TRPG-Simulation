import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, cleanup } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router'
import { SessionLobbyPage } from './session-lobby-page'
import type { SessionResponse } from '../api/types'

const mockGetSession = vi.fn()
const mockStartSession = vi.fn()
const mockGetScenario = vi.fn()
const mockListSessionPlayers = vi.fn()
const mockNavigate = vi.fn()

vi.mock('../api/sessions', () => ({
  getSession: (...args: unknown[]) => mockGetSession(...args),
  startSession: (...args: unknown[]) => mockStartSession(...args),
  listSessionPlayers: () => mockListSessionPlayers(),
}))

vi.mock('../api/scenarios', () => ({
  getScenario: (...args: unknown[]) => mockGetScenario(...args),
}))

vi.mock('react-router', async () => {
  const actual = await vi.importActual('react-router')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

// Mock auth store — default to GM user
const mockUseAuthStore = vi.fn()
vi.mock('../stores/auth-store', () => ({
  useAuthStore: (selector: (s: unknown) => unknown) => mockUseAuthStore(selector),
}))

function makeSession(overrides: Partial<SessionResponse> = {}): SessionResponse {
  return {
    id: 'session-1',
    scenarioId: 'sc-1',
    gmId: 'gm-user-id',
    status: 'lobby',
    inviteCode: 'ABC123',
    createdAt: '2026-03-01T12:00:00Z',
    startedAt: null,
    endedAt: null,
    ...overrides,
  }
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/sessions/session-1/lobby']}>
      <Routes>
        <Route path="/sessions/:id/lobby" element={<SessionLobbyPage />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('SessionLobbyPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockListSessionPlayers.mockResolvedValue({ players: [] })
    mockGetScenario.mockResolvedValue({
      id: 'sc-1',
      title: 'Test Scenario',
      status: 'published',
      content: {},
    })
  })

  afterEach(() => {
    cleanup()
  })

  it('GM sees Start Game button in lobby status', async () => {
    // GM user id matches session.gmId
    mockUseAuthStore.mockImplementation((selector: (s: { user: { id: string } }) => unknown) =>
      selector({ user: { id: 'gm-user-id' } }),
    )
    mockGetSession.mockResolvedValue(makeSession())

    renderPage()

    await waitFor(() => {
      expect(screen.getByText('Start Game')).toBeInTheDocument()
    })
  })

  it('Player sees waiting message in lobby status', async () => {
    // Player user id does NOT match session.gmId
    mockUseAuthStore.mockImplementation((selector: (s: { user: { id: string } }) => unknown) =>
      selector({ user: { id: 'player-user-id' } }),
    )
    mockGetSession.mockResolvedValue(makeSession())

    renderPage()

    await waitFor(() => {
      expect(
        screen.getByText('Waiting for GM to start the game...'),
      ).toBeInTheDocument()
    })
    // Player should NOT see Start Game button
    expect(screen.queryByText('Start Game')).not.toBeInTheDocument()
  })

  it('auto-navigates GM to console when status becomes active', async () => {
    mockUseAuthStore.mockImplementation((selector: (s: { user: { id: string } }) => unknown) =>
      selector({ user: { id: 'gm-user-id' } }),
    )
    // Return active session immediately
    mockGetSession.mockResolvedValue(makeSession({ status: 'active' }))

    renderPage()

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith(
        '/sessions/session-1/gm',
        { replace: true },
      )
    })
  })
})
