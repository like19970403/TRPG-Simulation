import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, cleanup } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router'
import { PlayerGuard } from './player-guard'

const mockGetSession = vi.fn()
vi.mock('../../api/sessions', () => ({
  getSession: (...args: unknown[]) => mockGetSession(...args),
}))

// Mock auth store — user is NOT the GM
const mockUser = { id: 'player-1', username: 'player_user' }
vi.mock('../../stores/auth-store', () => ({
  useAuthStore: (selector: (s: { user: typeof mockUser }) => unknown) =>
    selector({ user: mockUser }),
}))

function renderWithPlayerGuard(sessionId: string) {
  return render(
    <MemoryRouter initialEntries={[`/sessions/${sessionId}/play`]}>
      <Routes>
        <Route element={<PlayerGuard />}>
          <Route
            path="/sessions/:id/play"
            element={<div>Player Game Content</div>}
          />
        </Route>
        <Route
          path="/sessions/:id/gm"
          element={<div>GM Console (redirected)</div>}
        />
      </Routes>
    </MemoryRouter>,
  )
}

describe('PlayerGuard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  it('renders children when user is NOT the GM (player)', async () => {
    mockGetSession.mockResolvedValueOnce({
      id: 'session-1',
      gmId: 'other-user', // does NOT match mockUser.id
      scenarioId: 'scenario-1',
      status: 'active',
      inviteCode: 'ABC',
      createdAt: '2026-01-01T00:00:00Z',
      startedAt: null,
      endedAt: null,
    })

    renderWithPlayerGuard('session-1')

    await waitFor(() => {
      expect(screen.getByText('Player Game Content')).toBeInTheDocument()
    })
  })

  it('redirects GM to /sessions/:id/gm', async () => {
    mockGetSession.mockResolvedValueOnce({
      id: 'session-1',
      gmId: 'player-1', // matches mockUser.id — user IS the GM
      scenarioId: 'scenario-1',
      status: 'active',
      inviteCode: 'ABC',
      createdAt: '2026-01-01T00:00:00Z',
      startedAt: null,
      endedAt: null,
    })

    renderWithPlayerGuard('session-1')

    await waitFor(() => {
      expect(screen.getByText('GM Console (redirected)')).toBeInTheDocument()
    })
  })
})
