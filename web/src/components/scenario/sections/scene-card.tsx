import { useState } from 'react'
import { Input } from '../../ui/input'
import { Textarea } from '../../ui/textarea'
import { Button } from '../../ui/button'
import { TransitionEditor } from './transition-editor'
import { ActionEditor } from './action-editor'
import type { Scene, Transition, Action, Item, NPC } from '../../../api/types'
import { cn } from '../../../lib/cn'

interface SceneCardProps {
  scene: Scene
  onChange: (s: Scene) => void
  onRemove: () => void
  allSceneIds: string[]
  allItems: Item[]
  allNpcs: NPC[]
  allVariableNames: string[]
  defaultExpanded?: boolean
}

export function SceneCard({
  scene,
  onChange,
  onRemove,
  allSceneIds,
  allItems,
  allNpcs,
  allVariableNames,
  defaultExpanded = false,
}: SceneCardProps) {
  const [expanded, setExpanded] = useState(defaultExpanded)

  const allItemIds = allItems.map((i) => i.id)
  const allNpcIds = allNpcs.map((n) => n.id)
  const allNpcFieldKeys: Record<string, string[]> = {}
  for (const npc of allNpcs) {
    allNpcFieldKeys[npc.id] = (npc.fields ?? []).map((f) => f.key)
  }

  const updateTransition = (index: number, t: Transition) => {
    const transitions = [...(scene.transitions ?? [])]
    transitions[index] = t
    onChange({ ...scene, transitions })
  }

  const removeTransition = (index: number) => {
    const transitions = (scene.transitions ?? []).filter((_, i) => i !== index)
    onChange({ ...scene, transitions })
  }

  const addTransition = () => {
    const transitions = [
      ...(scene.transitions ?? []),
      { target: '', trigger: 'player_choice', label: '' },
    ]
    onChange({ ...scene, transitions })
  }

  const updateAction = (
    field: 'on_enter' | 'on_exit',
    index: number,
    a: Action,
  ) => {
    const actions = [...(scene[field] ?? [])]
    actions[index] = a
    onChange({ ...scene, [field]: actions })
  }

  const removeAction = (field: 'on_enter' | 'on_exit', index: number) => {
    const actions = (scene[field] ?? []).filter((_, i) => i !== index)
    onChange({ ...scene, [field]: actions })
  }

  const addAction = (field: 'on_enter' | 'on_exit') => {
    const actions = [
      ...(scene[field] ?? []),
      { set_var: { name: '', value: '' } },
    ]
    onChange({ ...scene, [field]: actions })
  }

  const toggleItem = (itemId: string) => {
    const current = scene.items_available ?? []
    const next = current.includes(itemId)
      ? current.filter((id) => id !== itemId)
      : [...current, itemId]
    onChange({ ...scene, items_available: next })
  }

  const toggleNpc = (npcId: string) => {
    const current = scene.npcs_present ?? []
    const next = current.includes(npcId)
      ? current.filter((id) => id !== npcId)
      : [...current, npcId]
    onChange({ ...scene, npcs_present: next })
  }

  return (
    <div className="rounded-lg border border-border bg-bg-card">
      {/* Header */}
      <div
        className="flex cursor-pointer items-center justify-between px-4 py-3"
        onClick={() => setExpanded(!expanded)}
      >
        <div className="flex items-center gap-2">
          <span className="text-xs text-text-tertiary">
            {expanded ? '▼' : '▶'}
          </span>
          <span className="text-sm font-medium text-text-primary">
            {scene.id || '（未命名）'}
          </span>
          {scene.name && (
            <span className="text-sm text-text-secondary">— {scene.name}</span>
          )}
        </div>
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation()
            onRemove()
          }}
          className="text-xs text-text-tertiary transition-colors hover:text-error"
        >
          刪除
        </button>
      </div>

      {/* Body */}
      {expanded && (
        <div className="flex flex-col gap-4 border-t border-border px-4 py-4">
          {/* ID + Name */}
          <div className="flex gap-3">
            <label className="flex flex-1 flex-col gap-1">
              <span className="text-xs font-medium text-text-secondary">
                場景 ID
              </span>
              <Input
                value={scene.id}
                onChange={(e) => onChange({ ...scene, id: e.target.value })}
                placeholder="場景 ID"
              />
            </label>
            <label className="flex flex-1 flex-col gap-1">
              <span className="text-xs font-medium text-text-secondary">
                名稱
              </span>
              <Input
                value={scene.name}
                onChange={(e) => onChange({ ...scene, name: e.target.value })}
                placeholder="場景名稱"
              />
            </label>
          </div>

          {/* Content */}
          <label className="flex flex-col gap-1">
            <span className="text-xs font-medium text-text-secondary">
              場景描述
            </span>
            <Textarea
              value={scene.content}
              onChange={(e) => onChange({ ...scene, content: e.target.value })}
              rows={6}
              placeholder="場景描述文字..."
            />
          </label>

          {/* GM Notes */}
          <label className="flex flex-col gap-1">
            <span className="text-xs font-medium text-text-secondary">
              GM 備註
            </span>
            <Textarea
              value={scene.gm_notes ?? ''}
              onChange={(e) =>
                onChange({
                  ...scene,
                  gm_notes: e.target.value || undefined,
                })
              }
              rows={3}
              placeholder="GM 專用備註..."
            />
          </label>

          {/* Items available */}
          {allItems.filter((i) => i.id).length > 0 && (
            <div className="flex flex-col gap-1">
              <span className="text-xs font-medium text-text-secondary">
                可用道具
              </span>
              <div className="flex flex-wrap gap-x-4 gap-y-1">
                {allItems.filter((i) => i.id).map((item) => (
                  <label
                    key={item.id}
                    className="flex items-center gap-1.5 text-sm text-text-primary"
                  >
                    <input
                      type="checkbox"
                      checked={(scene.items_available ?? []).includes(item.id)}
                      onChange={() => toggleItem(item.id)}
                      className={cn(
                        'h-3.5 w-3.5 rounded border-border bg-bg-input accent-gold',
                      )}
                    />
                    {item.id}
                    {item.name && (
                      <span className="text-text-tertiary">
                        — {item.name}
                      </span>
                    )}
                  </label>
                ))}
              </div>
            </div>
          )}

          {/* NPCs present */}
          {allNpcs.filter((n) => n.id).length > 0 && (
            <div className="flex flex-col gap-1">
              <span className="text-xs font-medium text-text-secondary">
                在場 NPC
              </span>
              <div className="flex flex-wrap gap-x-4 gap-y-1">
                {allNpcs.filter((n) => n.id).map((npc) => (
                  <label
                    key={npc.id}
                    className="flex items-center gap-1.5 text-sm text-text-primary"
                  >
                    <input
                      type="checkbox"
                      checked={(scene.npcs_present ?? []).includes(npc.id)}
                      onChange={() => toggleNpc(npc.id)}
                      className="h-3.5 w-3.5 rounded border-border bg-bg-input accent-gold"
                    />
                    {npc.id}
                    {npc.name && (
                      <span className="text-text-tertiary">
                        — {npc.name}
                      </span>
                    )}
                  </label>
                ))}
              </div>
            </div>
          )}

          {/* Transitions */}
          <div className="flex flex-col gap-2">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium text-text-secondary">
                場景轉換
              </span>
              <Button
                variant="secondary"
                size="sm"
                onClick={addTransition}
                type="button"
              >
                + 新增轉換
              </Button>
            </div>
            {(scene.transitions ?? []).length === 0 && (
              <p className="text-xs text-text-tertiary">無轉換（劇本結束）</p>
            )}
            {(scene.transitions ?? []).map((t, i) => (
              <TransitionEditor
                key={i}
                transition={t}
                onChange={(val) => updateTransition(i, val)}
                onRemove={() => removeTransition(i)}
                allSceneIds={allSceneIds}
                currentSceneId={scene.id}
              />
            ))}
          </div>

          {/* On Enter Actions */}
          <div className="flex flex-col gap-2">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium text-text-secondary">
                進入動作 (on_enter)
              </span>
              <Button
                variant="secondary"
                size="sm"
                onClick={() => addAction('on_enter')}
                type="button"
              >
                + 新增動作
              </Button>
            </div>
            {(scene.on_enter ?? []).length === 0 && (
              <p className="text-xs text-text-tertiary">無動作</p>
            )}
            {(scene.on_enter ?? []).map((a, i) => (
              <ActionEditor
                key={i}
                action={a}
                onChange={(val) => updateAction('on_enter', i, val)}
                onRemove={() => removeAction('on_enter', i)}
                allItemIds={allItemIds}
                allNpcIds={allNpcIds}
                allNpcFieldKeys={allNpcFieldKeys}
                allVariableNames={allVariableNames}
              />
            ))}
          </div>

          {/* On Exit Actions */}
          <div className="flex flex-col gap-2">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium text-text-secondary">
                離開動作 (on_exit)
              </span>
              <Button
                variant="secondary"
                size="sm"
                onClick={() => addAction('on_exit')}
                type="button"
              >
                + 新增動作
              </Button>
            </div>
            {(scene.on_exit ?? []).length === 0 && (
              <p className="text-xs text-text-tertiary">無動作</p>
            )}
            {(scene.on_exit ?? []).map((a, i) => (
              <ActionEditor
                key={i}
                action={a}
                onChange={(val) => updateAction('on_exit', i, val)}
                onRemove={() => removeAction('on_exit', i)}
                allItemIds={allItemIds}
                allNpcIds={allNpcIds}
                allNpcFieldKeys={allNpcFieldKeys}
                allVariableNames={allVariableNames}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
