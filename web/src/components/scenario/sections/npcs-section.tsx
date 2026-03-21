import { useRef } from 'react'
import { Button } from '../../ui/button'
import { NpcCard } from './npc-card'
import { generateNextId } from '../../../lib/scenario-id'
import type { NPC, Item } from '../../../api/types'

interface NpcsSectionProps {
  npcs: NPC[]
  onChange: (npcs: NPC[]) => void
  system?: string
  allItems?: Item[]
}

export function NpcsSection({ npcs, onChange, system, allItems }: NpcsSectionProps) {
  const newIndexRef = useRef<number | null>(null)

  const updateNpc = (index: number, npc: NPC) => {
    const next = [...npcs]
    next[index] = npc
    onChange(next)
  }

  const removeNpc = (index: number) => {
    onChange(npcs.filter((_, i) => i !== index))
  }

  const addNpc = () => {
    const newId = generateNextId('npc', npcs.map((n) => n.id))
    newIndexRef.current = npcs.length
    onChange([...npcs, { id: newId, name: '', fields: [] }])
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-text-secondary">
          NPC 列表
        </span>
        <Button
          variant="secondary"
          size="sm"
          onClick={addNpc}
          type="button"
        >
          + 新增 NPC
        </Button>
      </div>

      {npcs.length === 0 && (
        <p className="text-sm text-text-tertiary">尚未新增 NPC</p>
      )}

      {npcs.map((npc, i) => (
        <NpcCard
          key={i}
          npc={npc}
          onChange={(val) => updateNpc(i, val)}
          onRemove={() => removeNpc(i)}
          defaultExpanded={newIndexRef.current === i}
          system={system}
          allItems={allItems}
        />
      ))}
    </div>
  )
}
