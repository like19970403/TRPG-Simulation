import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { render, screen, waitFor, cleanup } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { LoginPage } from './login-page'

// Mock useAuth hook
const mockLogin = vi.fn()
vi.mock('../hooks/use-auth', () => ({
  useAuth: () => ({
    login: mockLogin,
    user: null,
    isAuthenticated: false,
    register: vi.fn(),
    logout: vi.fn(),
    tryRefresh: vi.fn(),
  }),
}))

// Mock react-router navigate
const mockNavigate = vi.fn()
vi.mock('react-router', async () => {
  const actual = await vi.importActual('react-router')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

function renderLoginPage() {
  return render(
    <MemoryRouter>
      <LoginPage />
    </MemoryRouter>,
  )
}

describe('LoginPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  it('renders sign in form', () => {
    renderLoginPage()
    expect(screen.getByText('Sign in to your account')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('you@example.com')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('••••••••')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument()
  })

  it('shows validation error for empty fields', async () => {
    renderLoginPage()
    const user = userEvent.setup()

    await user.click(screen.getByRole('button', { name: /sign in/i }))

    expect(screen.getByText('Please enter a valid email address')).toBeInTheDocument()
    expect(screen.getByText('Password is required')).toBeInTheDocument()
    expect(mockLogin).not.toHaveBeenCalled()
  })

  it('shows validation error for invalid email', async () => {
    renderLoginPage()
    const user = userEvent.setup()

    // Use an email-like value that passes HTML5 type="email" validation
    // but fails our stricter regex (missing TLD dot)
    await user.type(screen.getByPlaceholderText('you@example.com'), 'bad@email')
    await user.type(screen.getByPlaceholderText('••••••••'), 'password123')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    expect(screen.getByText('Please enter a valid email address')).toBeInTheDocument()
    expect(mockLogin).not.toHaveBeenCalled()
  })

  it('calls login and navigates on success', async () => {
    mockLogin.mockResolvedValueOnce(undefined)
    renderLoginPage()
    const user = userEvent.setup()

    await user.type(screen.getByPlaceholderText('you@example.com'), 'test@example.com')
    await user.type(screen.getByPlaceholderText('••••••••'), 'password123')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    await waitFor(() => {
      expect(mockLogin).toHaveBeenCalledWith({
        email: 'test@example.com',
        password: 'password123',
      })
    })
    expect(mockNavigate).toHaveBeenCalledWith('/dashboard')
  })

  it('shows server error on login failure', async () => {
    const { ApiClientError } = await import('../api/client')
    mockLogin.mockRejectedValueOnce(
      new ApiClientError(401, {
        error: 'INVALID_CREDENTIALS',
        message: 'Invalid email or password',
      }),
    )
    renderLoginPage()
    const user = userEvent.setup()

    await user.type(screen.getByPlaceholderText('you@example.com'), 'test@example.com')
    await user.type(screen.getByPlaceholderText('••••••••'), 'password123')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    await waitFor(() => {
      expect(screen.getByText('Invalid email or password')).toBeInTheDocument()
    })
  })

  it('has link to register page', () => {
    renderLoginPage()
    expect(screen.getByText('Create one')).toBeInTheDocument()
  })
})
