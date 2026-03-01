import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import { ItemDetailModal } from './item-detail-modal'
import type { Item } from '../../api/types'

afterEach(() => {
  cleanup()
})

const mockItem: Item = {
  id: 'item-1',
  name: 'Ancient Key',
  type: 'key_item',
  description: 'A rusty key that opens the crypt door.',
  image: 'https://example.com/key.png',
}

describe('ItemDetailModal', () => {
  it('renders nothing when open is false', () => {
    const { container } = render(
      <ItemDetailModal item={mockItem} open={false} onClose={vi.fn()} />,
    )
    expect(container.innerHTML).toBe('')
  })

  it('renders item details when open', () => {
    render(
      <ItemDetailModal item={mockItem} open={true} onClose={vi.fn()} />,
    )

    expect(screen.getByText('Ancient Key')).toBeInTheDocument()
    expect(screen.getByText('key_item')).toBeInTheDocument()
    expect(
      screen.getByText('A rusty key that opens the crypt door.'),
    ).toBeInTheDocument()
  })
})
