import { Input } from '../../ui/input'
import { Select } from '../../ui/select'
import { Button } from '../../ui/button'
import type { ScenarioVariable } from '../../../api/types'

interface VariablesSectionProps {
  variables: ScenarioVariable[]
  onChange: (variables: ScenarioVariable[]) => void
}

function getDefaultForType(type: string): unknown {
  switch (type) {
    case 'bool':
      return false
    case 'int':
      return 0
    default:
      return ''
  }
}

export function VariablesSection({
  variables,
  onChange,
}: VariablesSectionProps) {
  const updateVariable = (index: number, v: ScenarioVariable) => {
    const next = [...variables]
    next[index] = v
    onChange(next)
  }

  const removeVariable = (index: number) => {
    onChange(variables.filter((_, i) => i !== index))
  }

  const addVariable = () => {
    onChange([...variables, { name: '', type: 'bool', default: false }])
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-text-secondary">
          變數列表
        </span>
        <Button
          variant="secondary"
          size="sm"
          onClick={addVariable}
          type="button"
        >
          + 新增變數
        </Button>
      </div>

      {variables.length === 0 && (
        <p className="text-sm text-text-tertiary">尚未新增變數</p>
      )}

      {variables.map((v, i) => (
        <div
          key={i}
          className="flex items-end gap-2 rounded-lg border border-border bg-bg-card px-4 py-3"
        >
          <label className="flex flex-1 flex-col gap-1">
            <span className="text-xs text-text-tertiary">名稱</span>
            <Input
              value={v.name}
              onChange={(e) => updateVariable(i, { ...v, name: e.target.value })}
              placeholder="變數名稱"
            />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-xs text-text-tertiary">類型</span>
            <Select
              value={v.type}
              onChange={(e) => {
                const newType = e.target.value
                updateVariable(i, {
                  ...v,
                  type: newType,
                  default: getDefaultForType(newType),
                })
              }}
              className="w-28"
            >
              <option value="bool">bool</option>
              <option value="int">int</option>
              <option value="string">string</option>
            </Select>
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-xs text-text-tertiary">預設值</span>
            {v.type === 'bool' ? (
              <div className="flex h-[38px] items-center px-1">
                <input
                  type="checkbox"
                  checked={!!v.default}
                  onChange={(e) =>
                    updateVariable(i, { ...v, default: e.target.checked })
                  }
                  className="h-4 w-4 accent-gold"
                />
              </div>
            ) : v.type === 'int' ? (
              <Input
                type="number"
                value={String(v.default ?? 0)}
                onChange={(e) =>
                  updateVariable(i, {
                    ...v,
                    default: parseInt(e.target.value) || 0,
                  })
                }
                className="w-24"
              />
            ) : (
              <Input
                value={String(v.default ?? '')}
                onChange={(e) =>
                  updateVariable(i, { ...v, default: e.target.value })
                }
                placeholder="預設值"
                className="w-32"
              />
            )}
          </label>
          <button
            type="button"
            onClick={() => removeVariable(i)}
            className="shrink-0 pb-2.5 text-sm text-text-tertiary transition-colors hover:text-error"
          >
            刪除
          </button>
        </div>
      ))}
    </div>
  )
}
