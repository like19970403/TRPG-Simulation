import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ConfirmModal } from './confirm-modal'

afterEach(() => {
  cleanup()
})

describe('ConfirmModal', () => {
  it('renders nothing when open is false', () => {
    const { container } = render(
      <ConfirmModal
        open={false}
        onClose={vi.fn()}
        onConfirm={vi.fn()}
        title="Test"
        description="Test description"
        confirmLabel="OK"
      />,
    )
    expect(container.innerHTML).toBe('')
  })

  it('renders title, description, and buttons when open', () => {
    render(
      <ConfirmModal
        open={true}
        onClose={vi.fn()}
        onConfirm={vi.fn()}
        title="Delete Item?"
        description="This cannot be undone."
        confirmLabel="Delete"
      />,
    )
    expect(screen.getByText('Delete Item?')).toBeInTheDocument()
    expect(screen.getByText('This cannot be undone.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Delete' })).toBeInTheDocument()
  })

  it('calls onConfirm and onClose on button clicks', async () => {
    const onClose = vi.fn()
    const onConfirm = vi.fn()
    const user = userEvent.setup()

    render(
      <ConfirmModal
        open={true}
        onClose={onClose}
        onConfirm={onConfirm}
        title="Confirm?"
        description="Are you sure?"
        confirmLabel="Yes"
      />,
    )

    await user.click(screen.getByRole('button', { name: 'Yes' }))
    expect(onConfirm).toHaveBeenCalledTimes(1)

    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
