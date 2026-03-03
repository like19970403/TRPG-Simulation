import { useState } from 'react'
import { Input } from '../../ui/input'
import { Select } from '../../ui/select'
import type { Action, Item, NPC } from '../../../api/types'
import {
  ACTION_TYPE_LABELS,
  REVEAL_TARGET_LABELS,
  ARITHMETIC_OPERATORS,
} from '../../../lib/scenario-labels'

type ActionType = 'set_var' | 'reveal_item' | 'give_item' | 'remove_item' | 'reveal_npc_field'

interface ActionEditorProps {
  action: Action
  onChange: (a: Action) => void
  onRemove: () => void
  allItems: Item[]
  allNpcs: NPC[]
  allNpcFieldKeys: Record<string, string[]>
  allVariableNames: string[]
}

function getActionType(action: Action): ActionType {
  if (action.set_var) return 'set_var'
  if (action.reveal_item) return 'reveal_item'
  if (action.give_item) return 'give_item'
  if (action.remove_item) return 'remove_item'
  return 'reveal_npc_field'
}

/** Parse "varName op operand" from a set_var value string */
function parseArithExpr(varName: string, val: string) {
  const s = String(val).trim()
  const pattern = new RegExp(
    `^${varName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\s*([+\\-*/])\\s*(.+)$`,
  )
  const m = s.match(pattern)
  if (!m) return null
  return { op: m[1], operand: m[2] }
}

function makeDefaultAction(type: ActionType): Action {
  switch (type) {
    case 'set_var':
      return { set_var: { name: '', value: '' } }
    case 'reveal_item':
      return { reveal_item: { item_id: '', to: 'current_player' } }
    case 'give_item':
      return { give_item: { item_id: '', to: 'current_player' } }
    case 'remove_item':
      return { remove_item: { item_id: '', from: 'current_player' } }
    case 'reveal_npc_field':
      return {
        reveal_npc_field: {
          npc_id: '',
          field_key: '',
          to: 'current_player',
        },
      }
  }
}

export function ActionEditor({
  action,
  onChange,
  onRemove,
  allItems,
  allNpcs,
  allNpcFieldKeys,
  allVariableNames,
}: ActionEditorProps) {
  const actionType = getActionType(action)

  const handleTypeChange = (newType: ActionType) => {
    if (newType !== actionType) {
      onChange(makeDefaultAction(newType))
    }
  }

  return (
    <div className="flex flex-col gap-2 rounded-lg border border-border bg-[#1A1A1A] p-3">
      <div className="flex items-end gap-2">
        <label className="flex flex-col gap-1">
          <span className="text-xs text-text-tertiary">類型</span>
          <Select
            value={actionType}
            onChange={(e) => handleTypeChange(e.target.value as ActionType)}
            className="w-44"
          >
            {Object.entries(ACTION_TYPE_LABELS).map(([val, label]) => (
              <option key={val} value={val}>
                {label}
              </option>
            ))}
          </Select>
        </label>

        <div className="flex flex-1 gap-2">
          {actionType === 'set_var' && action.set_var && (
            <>
              <label className="flex flex-1 flex-col gap-1">
                <span className="text-xs text-text-tertiary">變數名稱</span>
                <Select
                  value={action.set_var.name}
                  onChange={(e) =>
                    onChange({
                      set_var: { ...action.set_var!, name: e.target.value },
                    })
                  }
                >
                  <option value="">-- 選擇變數 --</option>
                  {allVariableNames.map((n) => (
                    <option key={n} value={n}>
                      {n}
                    </option>
                  ))}
                </Select>
              </label>
              <SetVarValueBuilder
                varName={action.set_var.name}
                value={action.set_var.value}
                onChange={(newVal) =>
                  onChange({
                    set_var: { ...action.set_var!, value: newVal },
                  })
                }
              />
            </>
          )}

          {actionType === 'reveal_item' && action.reveal_item && (
            <>
              <label className="flex flex-1 flex-col gap-1">
                <span className="text-xs text-text-tertiary">道具</span>
                <Select
                  value={action.reveal_item.item_id}
                  onChange={(e) =>
                    onChange({
                      reveal_item: {
                        ...action.reveal_item!,
                        item_id: e.target.value,
                      },
                    })
                  }
                >
                  <option value="">-- 選擇道具 --</option>
                  {allItems.filter((i) => i.id).map((item) => (
                    <option key={item.id} value={item.id}>
                      {item.name || item.id}
                    </option>
                  ))}
                </Select>
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-xs text-text-tertiary">對象</span>
                <Select
                  value={action.reveal_item.to}
                  onChange={(e) =>
                    onChange({
                      reveal_item: {
                        ...action.reveal_item!,
                        to: e.target.value,
                      },
                    })
                  }
                  className="w-40"
                >
                  {Object.entries(REVEAL_TARGET_LABELS).map(([val, label]) => (
                    <option key={val} value={val}>
                      {label}
                    </option>
                  ))}
                </Select>
              </label>
            </>
          )}

          {actionType === 'give_item' && action.give_item && (
            <>
              <label className="flex flex-1 flex-col gap-1">
                <span className="text-xs text-text-tertiary">道具</span>
                <Select
                  value={action.give_item.item_id}
                  onChange={(e) =>
                    onChange({
                      give_item: {
                        ...action.give_item!,
                        item_id: e.target.value,
                      },
                    })
                  }
                >
                  <option value="">-- 選擇道具 --</option>
                  {allItems.filter((i) => i.id).map((item) => (
                    <option key={item.id} value={item.id}>
                      {item.name || item.id}
                    </option>
                  ))}
                </Select>
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-xs text-text-tertiary">對象</span>
                <Select
                  value={action.give_item.to}
                  onChange={(e) =>
                    onChange({
                      give_item: {
                        ...action.give_item!,
                        to: e.target.value,
                      },
                    })
                  }
                  className="w-40"
                >
                  {Object.entries(REVEAL_TARGET_LABELS).map(([val, label]) => (
                    <option key={val} value={val}>
                      {label}
                    </option>
                  ))}
                </Select>
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-xs text-text-tertiary">數量</span>
                <Input
                  type="number"
                  min={1}
                  value={action.give_item.quantity ?? 1}
                  onChange={(e) =>
                    onChange({
                      give_item: {
                        ...action.give_item!,
                        quantity: Math.max(1, parseInt(e.target.value) || 1),
                      },
                    })
                  }
                  className="w-20"
                />
              </label>
            </>
          )}

          {actionType === 'remove_item' && action.remove_item && (
            <>
              <label className="flex flex-1 flex-col gap-1">
                <span className="text-xs text-text-tertiary">道具</span>
                <Select
                  value={action.remove_item.item_id}
                  onChange={(e) =>
                    onChange({
                      remove_item: {
                        ...action.remove_item!,
                        item_id: e.target.value,
                      },
                    })
                  }
                >
                  <option value="">-- 選擇道具 --</option>
                  {allItems.filter((i) => i.id).map((item) => (
                    <option key={item.id} value={item.id}>
                      {item.name || item.id}
                    </option>
                  ))}
                </Select>
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-xs text-text-tertiary">對象</span>
                <Select
                  value={action.remove_item.from}
                  onChange={(e) =>
                    onChange({
                      remove_item: {
                        ...action.remove_item!,
                        from: e.target.value,
                      },
                    })
                  }
                  className="w-40"
                >
                  {Object.entries(REVEAL_TARGET_LABELS).map(([val, label]) => (
                    <option key={val} value={val}>
                      {label}
                    </option>
                  ))}
                </Select>
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-xs text-text-tertiary">數量</span>
                <Input
                  type="number"
                  min={0}
                  value={action.remove_item.quantity ?? 1}
                  onChange={(e) =>
                    onChange({
                      remove_item: {
                        ...action.remove_item!,
                        quantity: Math.max(0, parseInt(e.target.value) || 0),
                      },
                    })
                  }
                  className="w-20"
                />
              </label>
            </>
          )}

          {actionType === 'reveal_npc_field' && action.reveal_npc_field && (
            <>
              <label className="flex flex-1 flex-col gap-1">
                <span className="text-xs text-text-tertiary">NPC</span>
                <Select
                  value={action.reveal_npc_field.npc_id}
                  onChange={(e) =>
                    onChange({
                      reveal_npc_field: {
                        ...action.reveal_npc_field!,
                        npc_id: e.target.value,
                        field_key: '',
                      },
                    })
                  }
                >
                  <option value="">-- 選擇 NPC --</option>
                  {allNpcs.filter((n) => n.id).map((npc) => (
                    <option key={npc.id} value={npc.id}>
                      {npc.name || npc.id}
                    </option>
                  ))}
                </Select>
              </label>
              <label className="flex flex-1 flex-col gap-1">
                <span className="text-xs text-text-tertiary">欄位 Key</span>
                <Select
                  value={action.reveal_npc_field.field_key}
                  onChange={(e) =>
                    onChange({
                      reveal_npc_field: {
                        ...action.reveal_npc_field!,
                        field_key: e.target.value,
                      },
                    })
                  }
                >
                  <option value="">-- 選擇欄位 --</option>
                  {(
                    allNpcFieldKeys[action.reveal_npc_field.npc_id] ?? []
                  ).map((k) => (
                    <option key={k} value={k}>
                      {k}
                    </option>
                  ))}
                </Select>
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-xs text-text-tertiary">對象</span>
                <Select
                  value={action.reveal_npc_field.to}
                  onChange={(e) =>
                    onChange({
                      reveal_npc_field: {
                        ...action.reveal_npc_field!,
                        to: e.target.value,
                      },
                    })
                  }
                  className="w-40"
                >
                  {Object.entries(REVEAL_TARGET_LABELS).map(([val, label]) => (
                    <option key={val} value={val}>
                      {label}
                    </option>
                  ))}
                </Select>
              </label>
            </>
          )}
        </div>

        <button
          type="button"
          onClick={onRemove}
          className="shrink-0 pb-2.5 text-sm text-text-tertiary transition-colors hover:text-error"
        >
          刪除
        </button>
      </div>
    </div>
  )
}

/** Inline expression builder for set_var value: [operator ▼] [operand] with advanced toggle */
function SetVarValueBuilder({
  varName,
  value,
  onChange,
}: {
  varName: string
  value: unknown
  onChange: (v: string) => void
}) {
  const strVal = String(value ?? '')
  const parsed = varName ? parseArithExpr(varName, strVal) : null
  const canBeSimple = !strVal || parsed !== null

  const [advanced, setAdvanced] = useState(!canBeSimple)
  const [localOp, setLocalOp] = useState(parsed?.op ?? '')
  const [localOperand, setLocalOperand] = useState(parsed?.operand ?? strVal)

  const emit = (op: string, operand: string) => {
    if (!op) {
      // Direct value assignment
      onChange(operand)
    } else if (varName && operand) {
      onChange(`${varName} ${op} ${operand}`)
    }
  }

  const handleOpChange = (newOp: string) => {
    setLocalOp(newOp)
    emit(newOp, localOperand)
  }

  const handleOperandChange = (newOperand: string) => {
    setLocalOperand(newOperand)
    emit(localOp, newOperand)
  }

  if (advanced) {
    return (
      <label className="flex flex-1 flex-col gap-1">
        <span className="text-xs text-text-tertiary">值</span>
        <div className="flex gap-1">
          <Input
            value={strVal}
            onChange={(e) => onChange(e.target.value)}
            placeholder="expr 表達式"
            className="flex-1"
          />
          <button
            type="button"
            onClick={() => {
              const p = varName ? parseArithExpr(varName, strVal) : null
              if (p) {
                setLocalOp(p.op)
                setLocalOperand(p.operand)
              } else {
                setLocalOp('')
                setLocalOperand(strVal)
              }
              setAdvanced(false)
            }}
            className="shrink-0 text-xs text-text-tertiary transition-colors hover:text-gold"
            title="切換至簡易模式"
          >
            簡易
          </button>
        </div>
      </label>
    )
  }

  return (
    <div className="flex flex-1 items-end gap-1">
      <label className="flex flex-col gap-1">
        <span className="text-xs text-text-tertiary">運算</span>
        <Select
          value={localOp}
          onChange={(e) => handleOpChange(e.target.value)}
          className="w-28"
        >
          {ARITHMETIC_OPERATORS.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </Select>
      </label>
      <label className="flex flex-1 flex-col gap-1">
        <span className="text-xs text-text-tertiary">值</span>
        <Input
          value={localOperand}
          onChange={(e) => handleOperandChange(e.target.value)}
          placeholder={localOp ? '運算數值' : '直接設定的值'}
        />
      </label>
      <button
        type="button"
        onClick={() => setAdvanced(true)}
        className="shrink-0 pb-2.5 text-xs text-text-tertiary transition-colors hover:text-gold"
        title="切換至進階模式（可輸入任意表達式）"
      >
        進階
      </button>
    </div>
  )
}
