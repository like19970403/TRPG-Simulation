import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router'
import { AuthGuard } from './auth-guard'
import { useAuthStore } from '../stores/auth-store'

// Create a valid JWT
function makeToken(sub: string, username: string): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const payload = btoa(JSON.stringify({ sub, username }))
  return `${header}.${payload}.fake-signature`
}

// Mock the refresh API
vi.mock('../api/auth', () => ({
  refresh: vi.fn(),
}))

function renderWithRouter(initialPath: string) {
  return render(
    <MemoryRouter initialEntries={[initialPath]}>
      <Routes>
        <Route element={<AuthGuard />}>
          <Route path="/dashboard" element={<div>Dashboard Content</div>} />
        </Route>
        <Route path="/login" element={<div>Login Page</div>} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('AuthGuard', () => {
  beforeEach(() => {
    useAuthStore.getState().clearAuth()
    vi.clearAllMocks()
  })

  it('renders children when authenticated', () => {
    useAuthStore.getState().setAuth(makeToken('u1', 'aragorn'))
    renderWithRouter('/dashboard')
    expect(screen.getByText('Dashboard Content')).toBeInTheDocument()
  })

  it('redirects to login when not authenticated and refresh fails', async () => {
    const { refresh } = await import('../api/auth')
    vi.mocked(refresh).mockRejectedValueOnce(new Error('no token'))

    renderWithRouter('/dashboard')

    await waitFor(() => {
      expect(screen.getByText('Login Page')).toBeInTheDocument()
    })
  })

  it('renders children after successful refresh', async () => {
    const { refresh } = await import('../api/auth')
    const token = makeToken('u1', 'aragorn')
    vi.mocked(refresh).mockResolvedValueOnce({
      accessToken: token,
      expiresIn: 900,
      tokenType: 'Bearer',
    })

    renderWithRouter('/dashboard')

    await waitFor(() => {
      expect(screen.getByText('Dashboard Content')).toBeInTheDocument()
    })
  })
})
