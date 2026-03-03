import { useRef } from 'react'
import { Button } from '../../ui/button'
import { ItemCard } from './item-card'
import { generateNextId } from '../../../lib/scenario-id'
import type { Item } from '../../../api/types'

interface ItemsSectionProps {
  items: Item[]
  onChange: (items: Item[]) => void
}

export function ItemsSection({ items, onChange }: ItemsSectionProps) {
  const newIndexRef = useRef<number | null>(null)

  const updateItem = (index: number, item: Item) => {
    const next = [...items]
    next[index] = item
    onChange(next)
  }

  const removeItem = (index: number) => {
    onChange(items.filter((_, i) => i !== index))
  }

  const addItem = () => {
    const newId = generateNextId('item', items.map((i) => i.id))
    newIndexRef.current = items.length
    onChange([
      ...items,
      { id: newId, name: '', type: 'item', description: '' },
    ])
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-text-secondary">
          道具列表
        </span>
        <Button
          variant="secondary"
          size="sm"
          onClick={addItem}
          type="button"
        >
          + 新增道具
        </Button>
      </div>

      {items.length === 0 && (
        <p className="text-sm text-text-tertiary">尚未新增道具</p>
      )}

      {items.map((item, i) => (
        <ItemCard
          key={i}
          item={item}
          onChange={(val) => updateItem(i, val)}
          onRemove={() => removeItem(i)}
          defaultExpanded={newIndexRef.current === i}
        />
      ))}
    </div>
  )
}
