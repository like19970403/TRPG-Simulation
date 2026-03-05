import { useState, useCallback } from 'react'
import { Input } from '../../ui/input'
import { Select } from '../../ui/select'
import type { ScenarioVariable, Item } from '../../../api/types'
import { CONDITION_OPERATORS } from '../../../lib/scenario-labels'
import { HelpIcon } from '../../ui/tooltip'

interface ConditionBuilderProps {
  value: string
  onChange: (expr: string) => void
  allVariables: ScenarioVariable[]
  allItems?: Item[]
}

type ConditionType = 'variable' | 'has_item'

/** Try to parse a simple "varName op value" expression */
function parseSimple(expr: string) {
  const m = expr.trim().match(/^(\w+)\s*(==|!=|>=|<=|>|<)\s*(.+)$/)
  if (!m) return null
  return { varName: m[1], op: m[2], val: m[3] }
}

/** Try to parse has_item("itemId") or has_item('itemId') */
function parseHasItem(expr: string): string | null {
  const m = expr.trim().match(/^has_item\(["'](.+?)["']\)$/)
  return m ? m[1] : null
}

/** Detect the condition type from an expression string */
function detectType(expr: string): ConditionType {
  if (parseHasItem(expr) !== null) return 'has_item'
  return 'variable'
}

function buildExpr(v: string, o: string, va: string) {
  if (!v) return ''
  return va ? `${v} ${o} ${va}` : ''
}

export function ConditionBuilder({
  value,
  onChange,
  allVariables,
  allItems = [],
}: ConditionBuilderProps) {
  const parsed = parseSimple(value)
  const hasItemId = parseHasItem(value)
  const canBeSimple = !value || parsed !== null || hasItemId !== null

  const [advanced, setAdvanced] = useState(!canBeSimple)
  const [condType, setCondType] = useState<ConditionType>(
    value ? detectType(value) : 'variable',
  )

  // Variable comparison state
  const [localVar, setLocalVar] = useState(parsed?.varName ?? '')
  const [localOp, setLocalOp] = useState(parsed?.op ?? '==')
  const [localVal, setLocalVal] = useState(parsed?.val ?? '')

  // Item check state
  const [localItemId, setLocalItemId] = useState(hasItemId ?? '')

  // Sync from prop when it changes externally (e.g. loading a saved scenario)
  const varName = advanced ? '' : (parsed?.varName ?? localVar)
  const op = advanced ? '==' : (parsed?.op ?? localOp)
  const val = advanced ? '' : (parsed?.val ?? localVal)

  const selectedVar = allVariables.find(
    (v) => v.name === (advanced ? '' : (localVar || varName)),
  )

  const emitChange = useCallback(
    (v: string, o: string, va: string) => {
      const expr = buildExpr(v, o, va)
      onChange(expr)
    },
    [onChange],
  )

  const handleCondTypeChange = (newType: ConditionType) => {
    setCondType(newType)
    if (newType === 'has_item') {
      if (localItemId) {
        onChange(`has_item("${localItemId}")`)
      } else {
        onChange('')
      }
    } else {
      emitChange(localVar, localOp, localVal)
    }
  }

  const handleItemChange = (itemId: string) => {
    setLocalItemId(itemId)
    if (itemId) {
      onChange(`has_item("${itemId}")`)
    } else {
      onChange('')
    }
  }

  const handleVarChange = (newVar: string) => {
    setLocalVar(newVar)
    const sv = allVariables.find((v) => v.name === newVar)
    const defaultVal =
      sv?.type === 'bool' ? 'true' : sv?.type === 'int' ? '0' : ''
    setLocalVal(defaultVal)
    emitChange(newVar, localOp, defaultVal)
  }

  const handleOpChange = (newOp: string) => {
    setLocalOp(newOp)
    emitChange(localVar, newOp, localVal)
  }

  const handleValChange = (newVal: string) => {
    setLocalVal(newVal)
    emitChange(localVar, localOp, newVal)
  }

  if (advanced) {
    return (
      <div className="flex gap-1">
        <Input
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="expr 條件表達式"
          className="flex-1"
        />
        <button
          type="button"
          onClick={() => {
            const hi = parseHasItem(value)
            if (hi !== null) {
              setCondType('has_item')
              setLocalItemId(hi)
            } else {
              setCondType('variable')
              const p = parseSimple(value)
              if (p) {
                setLocalVar(p.varName)
                setLocalOp(p.op)
                setLocalVal(p.val)
              } else {
                setLocalVar('')
                setLocalOp('==')
                setLocalVal('')
              }
            }
            setAdvanced(false)
          }}
          className="shrink-0 text-xs text-text-tertiary transition-colors hover:text-gold"
          title="切換至簡易模式"
        >
          簡易
        </button>
      </div>
    )
  }

  return (
    <div className="flex gap-1">
      {/* Condition type selector */}
      <Select
        value={condType}
        onChange={(e) =>
          handleCondTypeChange(e.target.value as ConditionType)
        }
        className="w-28"
      >
        <option value="variable">變數比較</option>
        <option value="has_item">持有道具</option>
      </Select>

      {condType === 'has_item' ? (
        /* Item check mode */
        <Select
          value={localItemId || (hasItemId ?? '')}
          onChange={(e) => handleItemChange(e.target.value)}
          className="flex-1"
        >
          <option value="">-- 選擇道具 --</option>
          {allItems
            .filter((i) => i.id)
            .map((item) => (
              <option key={item.id} value={item.id}>
                {item.name || item.id}
              </option>
            ))}
        </Select>
      ) : (
        /* Variable comparison mode */
        <>
          <Select
            value={localVar || varName}
            onChange={(e) => handleVarChange(e.target.value)}
            className="w-32"
          >
            <option value="">-- 變數 --</option>
            {allVariables
              .filter((v) => v.name)
              .map((v) => (
                <option key={v.name} value={v.name}>
                  {v.name}
                </option>
              ))}
          </Select>
          <Select
            value={localOp || op}
            onChange={(e) => handleOpChange(e.target.value)}
            className="w-32"
          >
            {CONDITION_OPERATORS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </Select>
          {selectedVar?.type === 'bool' ? (
            <Select
              value={localVal || val}
              onChange={(e) => handleValChange(e.target.value)}
              className="w-24"
            >
              <option value="true">true</option>
              <option value="false">false</option>
            </Select>
          ) : (
            <Input
              type={selectedVar?.type === 'int' ? 'number' : 'text'}
              value={localVal || val}
              onChange={(e) => handleValChange(e.target.value)}
              placeholder="值"
              className="w-24"
            />
          )}
        </>
      )}

      <button
        type="button"
        onClick={() => setAdvanced(true)}
        className="shrink-0 text-xs text-text-tertiary transition-colors hover:text-gold"
        title="切換至進階模式（可輸入複合條件）"
      >
        進階
      </button>
      <HelpIcon tip="支援 expr 語法：變數比較 (線索數 >= 10)、邏輯組合 (&&, ||, !)、道具檢查 has_item('item_id')、括號分組。" />
    </div>
  )
}
