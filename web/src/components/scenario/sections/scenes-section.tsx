import { useRef } from 'react'
import { Button } from '../../ui/button'
import { SceneCard } from './scene-card'
import { generateNextId } from '../../../lib/scenario-id'
import type { Scene, Item, NPC, ScenarioVariable } from '../../../api/types'

interface ScenesSectionProps {
  scenes: Scene[]
  onChange: (scenes: Scene[]) => void
  allSceneIds: string[]
  allItems: Item[]
  allNpcs: NPC[]
  allVariableNames: string[]
  allVariables: ScenarioVariable[]
}

export function ScenesSection({
  scenes,
  onChange,
  allSceneIds,
  allItems,
  allNpcs,
  allVariableNames,
  allVariables,
}: ScenesSectionProps) {
  const newIndexRef = useRef<number | null>(null)

  const updateScene = (index: number, s: Scene) => {
    const next = [...scenes]
    next[index] = s
    onChange(next)
  }

  const removeScene = (index: number) => {
    onChange(scenes.filter((_, i) => i !== index))
  }

  const addScene = () => {
    const newId = generateNextId('scene', scenes.map((s) => s.id))
    newIndexRef.current = scenes.length
    onChange([
      ...scenes,
      {
        id: newId,
        name: '',
        content: '',
        transitions: [],
      },
    ])
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-text-secondary">
          場景列表
        </span>
        <Button
          variant="secondary"
          size="sm"
          onClick={addScene}
          type="button"
        >
          + 新增場景
        </Button>
      </div>

      {scenes.length === 0 && (
        <p className="text-sm text-text-tertiary">尚未新增場景</p>
      )}

      {scenes.map((scene, i) => (
        <SceneCard
          key={i}
          scene={scene}
          onChange={(val) => updateScene(i, val)}
          onRemove={() => removeScene(i)}
          allSceneIds={allSceneIds}
          allScenes={scenes}
          allItems={allItems}
          allNpcs={allNpcs}
          allVariableNames={allVariableNames}
          allVariables={allVariables}
          defaultExpanded={newIndexRef.current === i}
        />
      ))}
    </div>
  )
}
