import { useState } from 'react'
import { useGameStore } from '../../stores/game-store'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import type { SendAction } from '../../hooks/use-game-socket'

const EMPTY_VARS: Record<string, unknown> = {}

interface VariablesPanelProps {
  sendAction: SendAction
}

export function VariablesPanel({ sendAction }: VariablesPanelProps) {
  const variables = useGameStore(
    (s) => s.gameState?.variables ?? EMPTY_VARS,
  )
  const scenarioVars = useGameStore(
    (s) => s.scenarioContent?.variables,
  )

  const [editingVar, setEditingVar] = useState<string | null>(null)
  const [editValue, setEditValue] = useState('')

  const entries = Object.entries(variables)

  // Find the scenario variable definition (for type info)
  const getVarDef = (name: string) =>
    scenarioVars?.find((v) => v.name === name)

  const startEdit = (name: string, currentValue: unknown) => {
    setEditingVar(name)
    setEditValue(String(currentValue ?? ''))
  }

  const commitEdit = (name: string) => {
    const def = getVarDef(name)
    let parsedValue: unknown = editValue

    if (def?.type === 'bool') {
      parsedValue = editValue === 'true'
    } else if (def?.type === 'int') {
      const n = Number(editValue)
      parsedValue = Number.isNaN(n) ? 0 : n
    }

    sendAction('set_variable', { name, value: parsedValue })
    setEditingVar(null)
    setEditValue('')
  }

  const formatValue = (value: unknown): string => {
    if (typeof value === 'boolean') return value ? 'true' : 'false'
    if (value === null || value === undefined) return '—'
    return String(value)
  }

  return (
    <div className="flex flex-1 flex-col overflow-y-auto p-3">
      {entries.length === 0 ? (
        <p className="text-xs text-text-tertiary">此劇本無定義變數</p>
      ) : (
        <div className="flex flex-col gap-1">
          {entries.map(([name, value]) => {
            const def = getVarDef(name)
            const isEditing = editingVar === name
            return (
              <div
                key={name}
                className="flex items-center gap-2 rounded px-2 py-1.5 text-xs hover:bg-bg-input"
              >
                <span className="min-w-0 shrink-0 font-mono text-text-tertiary">
                  {name}
                </span>
                {def && (
                  <span className="shrink-0 rounded bg-bg-input px-1 text-text-tertiary">
                    {def.type}
                  </span>
                )}
                <span className="mx-1 text-text-tertiary">=</span>
                {isEditing ? (
                  <form
                    className="flex flex-1 items-center gap-1"
                    onSubmit={(e) => {
                      e.preventDefault()
                      commitEdit(name)
                    }}
                  >
                    {def?.type === 'bool' ? (
                      <select
                        className="h-6 rounded border border-border bg-bg-input px-1 text-xs text-text-primary"
                        value={editValue}
                        onChange={(e) => setEditValue(e.target.value)}
                        autoFocus
                      >
                        <option value="true">true</option>
                        <option value="false">false</option>
                      </select>
                    ) : (
                      <Input
                        className="h-6 flex-1 px-1 text-xs"
                        type={def?.type === 'int' ? 'number' : 'text'}
                        value={editValue}
                        onChange={(e) => setEditValue(e.target.value)}
                        autoFocus
                      />
                    )}
                    <Button
                      type="submit"
                      variant="primary"
                      size="sm"
                      className="h-6 px-2 text-xs"
                    >
                      確認
                    </Button>
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="h-6 px-2 text-xs"
                      onClick={() => setEditingVar(null)}
                    >
                      取消
                    </Button>
                  </form>
                ) : (
                  <>
                    <span className="flex-1 font-mono text-text-primary">
                      {formatValue(value)}
                    </span>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-6 shrink-0 px-2 text-xs text-text-tertiary hover:text-gold"
                      onClick={() => startEdit(name, value)}
                    >
                      編輯
                    </Button>
                  </>
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
