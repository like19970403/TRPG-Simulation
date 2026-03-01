import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, cleanup } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router'
import { GmGuard } from './gm-guard'

const mockGetSession = vi.fn()
vi.mock('../../api/sessions', () => ({
  getSession: (...args: unknown[]) => mockGetSession(...args),
}))

// Mock auth store
const mockUser = { id: 'user-1', username: 'gm_user' }
vi.mock('../../stores/auth-store', () => ({
  useAuthStore: (selector: (s: { user: typeof mockUser }) => unknown) =>
    selector({ user: mockUser }),
}))

function renderWithGmGuard(sessionId: string) {
  return render(
    <MemoryRouter initialEntries={[`/sessions/${sessionId}/gm`]}>
      <Routes>
        <Route element={<GmGuard />}>
          <Route
            path="/sessions/:id/gm"
            element={<div>GM Console Content</div>}
          />
        </Route>
        <Route
          path="/sessions/:id"
          element={<div>Session Detail (redirected)</div>}
        />
      </Routes>
    </MemoryRouter>,
  )
}

describe('GmGuard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  it('renders children when user is GM', async () => {
    mockGetSession.mockResolvedValueOnce({
      id: 'session-1',
      gmId: 'user-1', // matches mockUser.id
      scenarioId: 'scenario-1',
      status: 'active',
      inviteCode: 'ABC',
      createdAt: '2026-01-01T00:00:00Z',
      startedAt: null,
      endedAt: null,
    })

    renderWithGmGuard('session-1')

    await waitFor(() => {
      expect(screen.getByText('GM Console Content')).toBeInTheDocument()
    })
  })

  it('redirects when user is not GM', async () => {
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

    renderWithGmGuard('session-1')

    await waitFor(() => {
      expect(
        screen.getByText('Session Detail (redirected)'),
      ).toBeInTheDocument()
    })
  })
})
