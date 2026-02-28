import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { render, screen, waitFor, cleanup } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { RegisterPage } from './register-page'

const mockRegister = vi.fn()
vi.mock('../hooks/use-auth', () => ({
  useAuth: () => ({
    register: mockRegister,
    login: vi.fn(),
    user: null,
    isAuthenticated: false,
    logout: vi.fn(),
    tryRefresh: vi.fn(),
  }),
}))

const mockNavigate = vi.fn()
vi.mock('react-router', async () => {
  const actual = await vi.importActual('react-router')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

function renderRegisterPage() {
  return render(
    <MemoryRouter>
      <RegisterPage />
    </MemoryRouter>,
  )
}

describe('RegisterPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  it('renders registration form', () => {
    renderRegisterPage()
    expect(screen.getByText('Create your account')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('adventurer_01')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('you@example.com')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Min 8 characters')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /create account/i })).toBeInTheDocument()
  })

  it('validates username length', async () => {
    renderRegisterPage()
    const user = userEvent.setup()

    await user.type(screen.getByPlaceholderText('adventurer_01'), 'ab')
    await user.type(screen.getByPlaceholderText('you@example.com'), 'test@example.com')
    await user.type(screen.getByPlaceholderText('Min 8 characters'), 'password123')
    await user.click(screen.getByRole('button', { name: /create account/i }))

    expect(screen.getByText('Must be between 3 and 50 characters')).toBeInTheDocument()
    expect(mockRegister).not.toHaveBeenCalled()
  })

  it('validates username characters', async () => {
    renderRegisterPage()
    const user = userEvent.setup()

    await user.type(screen.getByPlaceholderText('adventurer_01'), 'bad name!')
    await user.type(screen.getByPlaceholderText('you@example.com'), 'test@example.com')
    await user.type(screen.getByPlaceholderText('Min 8 characters'), 'password123')
    await user.click(screen.getByRole('button', { name: /create account/i }))

    expect(screen.getByText('Only letters, numbers, and underscores allowed')).toBeInTheDocument()
  })

  it('validates password length', async () => {
    renderRegisterPage()
    const user = userEvent.setup()

    await user.type(screen.getByPlaceholderText('adventurer_01'), 'valid_user')
    await user.type(screen.getByPlaceholderText('you@example.com'), 'test@example.com')
    await user.type(screen.getByPlaceholderText('Min 8 characters'), 'short')
    await user.click(screen.getByRole('button', { name: /create account/i }))

    expect(screen.getByText('Must be between 8 and 72 characters')).toBeInTheDocument()
  })

  it('calls register and navigates to login on success', async () => {
    mockRegister.mockResolvedValueOnce({ id: '1', username: 'gandalf', email: 'g@m.com', createdAt: '' })
    renderRegisterPage()
    const user = userEvent.setup()

    await user.type(screen.getByPlaceholderText('adventurer_01'), 'gandalf')
    await user.type(screen.getByPlaceholderText('you@example.com'), 'g@middle.earth')
    await user.type(screen.getByPlaceholderText('Min 8 characters'), 'youshallnotpass')
    await user.click(screen.getByRole('button', { name: /create account/i }))

    await waitFor(() => {
      expect(mockRegister).toHaveBeenCalledWith({
        username: 'gandalf',
        email: 'g@middle.earth',
        password: 'youshallnotpass',
      })
    })
    expect(mockNavigate).toHaveBeenCalledWith('/login')
  })

  it('shows server error on conflict', async () => {
    const { ApiClientError } = await import('../api/client')
    mockRegister.mockRejectedValueOnce(
      new ApiClientError(409, {
        error: 'CONFLICT',
        message: 'Username or email already exists',
      }),
    )
    renderRegisterPage()
    const user = userEvent.setup()

    await user.type(screen.getByPlaceholderText('adventurer_01'), 'gandalf')
    await user.type(screen.getByPlaceholderText('you@example.com'), 'g@middle.earth')
    await user.type(screen.getByPlaceholderText('Min 8 characters'), 'youshallnotpass')
    await user.click(screen.getByRole('button', { name: /create account/i }))

    await waitFor(() => {
      expect(screen.getByText('Username or email already exists')).toBeInTheDocument()
    })
  })

  it('has link to login page', () => {
    renderRegisterPage()
    expect(screen.getByText('Sign in')).toBeInTheDocument()
  })
})
