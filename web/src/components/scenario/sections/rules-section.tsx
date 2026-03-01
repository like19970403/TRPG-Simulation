import { Input } from '../../ui/input'
import { Select } from '../../ui/select'
import { Button } from '../../ui/button'
import type { Rules, Attribute } from '../../../api/types'

interface RulesSectionProps {
  rules: Rules | undefined
  onChange: (rules: Rules | undefined) => void
}

export function RulesSection({ rules, onChange }: RulesSectionProps) {
  const r = rules ?? {}

  const update = (patch: Partial<Rules>) => {
    onChange({ ...r, ...patch })
  }

  const updateAttribute = (index: number, attr: Attribute) => {
    const attrs = [...(r.attributes ?? [])]
    attrs[index] = attr
    update({ attributes: attrs })
  }

  const removeAttribute = (index: number) => {
    const attrs = (r.attributes ?? []).filter((_, i) => i !== index)
    update({ attributes: attrs })
  }

  const addAttribute = () => {
    const attrs = [
      ...(r.attributes ?? []),
      { name: '', display: '', default: 10 },
    ]
    update({ attributes: attrs })
  }

  return (
    <div className="flex flex-col gap-4">
      <p className="text-xs text-text-tertiary">
        規則設定（選填）— 定義骰子公式、檢定方式和角色屬性
      </p>

      <div className="flex gap-3">
        <label className="flex flex-1 flex-col gap-1">
          <span className="text-xs font-medium text-text-secondary">
            骰子公式
          </span>
          <Input
            value={r.dice_formula ?? ''}
            onChange={(e) =>
              update({ dice_formula: e.target.value || undefined })
            }
            placeholder="例如：2d6、d20+5"
          />
        </label>
        <label className="flex flex-col gap-1">
          <span className="text-xs font-medium text-text-secondary">
            檢定方式
          </span>
          <Select
            value={r.check_method ?? ''}
            onChange={(e) =>
              update({ check_method: e.target.value || undefined })
            }
            className="w-36"
          >
            <option value="">-- 無 --</option>
            <option value="gte">gte（大於等於）</option>
            <option value="gt">gt（大於）</option>
          </Select>
        </label>
      </div>

      {/* Attributes */}
      <div className="flex flex-col gap-2">
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium text-text-secondary">
            屬性列表
          </span>
          <Button
            variant="secondary"
            size="sm"
            onClick={addAttribute}
            type="button"
          >
            + 新增屬性
          </Button>
        </div>

        {(r.attributes ?? []).length === 0 && (
          <p className="text-xs text-text-tertiary">尚未定義屬性</p>
        )}

        {(r.attributes ?? []).map((attr, i) => (
          <div
            key={i}
            className="flex items-end gap-2 rounded-lg border border-border bg-bg-card px-4 py-3"
          >
            <label className="flex flex-1 flex-col gap-1">
              <span className="text-xs text-text-tertiary">Name</span>
              <Input
                value={attr.name}
                onChange={(e) =>
                  updateAttribute(i, { ...attr, name: e.target.value })
                }
                placeholder="屬性 key"
              />
            </label>
            <label className="flex flex-1 flex-col gap-1">
              <span className="text-xs text-text-tertiary">Display</span>
              <Input
                value={attr.display}
                onChange={(e) =>
                  updateAttribute(i, { ...attr, display: e.target.value })
                }
                placeholder="顯示名稱"
              />
            </label>
            <label className="flex flex-col gap-1">
              <span className="text-xs text-text-tertiary">Default</span>
              <Input
                type="number"
                value={String(attr.default)}
                onChange={(e) =>
                  updateAttribute(i, {
                    ...attr,
                    default: parseInt(e.target.value) || 0,
                  })
                }
                className="w-20"
              />
            </label>
            <button
              type="button"
              onClick={() => removeAttribute(i)}
              className="shrink-0 pb-2.5 text-sm text-text-tertiary transition-colors hover:text-error"
            >
              刪除
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}
