import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, cleanup } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { JoinSessionModal } from './join-session-modal'

const mockJoinSession = vi.fn()
vi.mock('../../api/sessions', () => ({
  joinSession: (...args: unknown[]) => mockJoinSession(...args),
}))

const mockNavigate = vi.fn()
vi.mock('react-router', async () => {
  const actual = await vi.importActual('react-router')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

function renderModal() {
  return render(
    <MemoryRouter>
      <JoinSessionModal open={true} onClose={vi.fn()} onJoined={vi.fn()} />
    </MemoryRouter>,
  )
}

describe('JoinSessionModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  it('calls joinSession and navigates to lobby on success', async () => {
    mockJoinSession.mockResolvedValueOnce({
      id: 'session-1',
      scenarioId: 'sc-1',
      gmId: 'gm-1',
      status: 'lobby',
      inviteCode: 'ABC123',
      createdAt: '2026-03-01T00:00:00Z',
      startedAt: null,
      endedAt: null,
    })

    const user = userEvent.setup()
    renderModal()

    await user.type(screen.getByPlaceholderText('Enter invite code'), 'ABC123')
    await user.click(screen.getByRole('button', { name: 'Join' }))

    await waitFor(() => {
      expect(mockJoinSession).toHaveBeenCalledWith({ inviteCode: 'ABC123' })
      expect(mockNavigate).toHaveBeenCalledWith('/sessions/session-1/lobby')
    })
  })

  it('shows error message on API failure', async () => {
    const { ApiClientError } = await import('../../api/client')
    mockJoinSession.mockRejectedValueOnce(
      new ApiClientError(404, {
        error: 'NOT_FOUND',
        message: 'Invalid invite code',
      }),
    )

    const user = userEvent.setup()
    renderModal()

    await user.type(screen.getByPlaceholderText('Enter invite code'), 'BADCODE')
    await user.click(screen.getByRole('button', { name: 'Join' }))

    await waitFor(() => {
      expect(screen.getByText('Invalid invite code')).toBeInTheDocument()
    })
  })
})
