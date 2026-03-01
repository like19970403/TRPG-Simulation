import { useState, useEffect } from 'react'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { Textarea } from '../ui/textarea'
import { createCharacter, updateCharacter } from '../../api/characters'
import { ApiClientError } from '../../api/client'
import type { CharacterResponse } from '../../api/types'
import { cn } from '../../lib/cn'

interface CharacterFormModalProps {
  open: boolean
  onClose: () => void
  onSaved: () => void
  character?: CharacterResponse | null
}

type AttrRow = { key: string; value: string }

function attrsToRows(attrs: Record<string, unknown>): AttrRow[] {
  const entries = Object.entries(attrs)
  return entries.length > 0
    ? entries.map(([key, value]) => ({ key, value: String(value ?? '') }))
    : []
}

function rowsToAttrs(rows: AttrRow[]): Record<string, unknown> {
  const result: Record<string, unknown> = {}
  for (const row of rows) {
    if (!row.key.trim()) continue
    const num = Number(row.value)
    if (row.value !== '' && !isNaN(num)) {
      result[row.key.trim()] = num
    } else if (row.value === 'true') {
      result[row.key.trim()] = true
    } else if (row.value === 'false') {
      result[row.key.trim()] = false
    } else {
      result[row.key.trim()] = row.value
    }
  }
  return result
}

function inventoryToList(inv: unknown[]): string[] {
  return inv.map((item) =>
    typeof item === 'string' ? item : JSON.stringify(item),
  )
}

function listToInventory(list: string[]): unknown[] {
  return list.filter((s) => s.trim()).map((s) => {
    try {
      const parsed = JSON.parse(s)
      return parsed
    } catch {
      return s
    }
  })
}

function tryParseJSON(text: string): { ok: true; value: unknown } | { ok: false } {
  try {
    return { ok: true, value: JSON.parse(text) }
  } catch {
    return { ok: false }
  }
}

export function CharacterFormModal({
  open,
  onClose,
  onSaved,
  character,
}: CharacterFormModalProps) {
  const isEdit = !!character

  const [name, setName] = useState('')
  const [notes, setNotes] = useState('')
  const [editorMode, setEditorMode] = useState<'form' | 'json'>('form')

  // Form mode state
  const [attrRows, setAttrRows] = useState<AttrRow[]>([])
  const [inventoryList, setInventoryList] = useState<string[]>([])

  // JSON mode state
  const [attributesText, setAttributesText] = useState('{}')
  const [inventoryText, setInventoryText] = useState('[]')

  const [switchError, setSwitchError] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (open && character) {
      setName(character.name)
      setNotes(character.notes)
      setAttrRows(attrsToRows(character.attributes))
      setInventoryList(inventoryToList(character.inventory))
      setAttributesText(JSON.stringify(character.attributes, null, 2))
      setInventoryText(JSON.stringify(character.inventory, null, 2))
      setEditorMode('form')
      setError('')
      setSwitchError('')
    } else if (open) {
      setName('')
      setNotes('')
      setAttrRows([])
      setInventoryList([])
      setAttributesText('{}')
      setInventoryText('[]')
      setEditorMode('form')
      setError('')
      setSwitchError('')
    }
  }, [open, character])

  if (!open) return null

  // JSON mode validation
  const attrsResult = tryParseJSON(attributesText)
  const invResult = tryParseJSON(inventoryText)
  const attrsValid =
    attrsResult.ok &&
    typeof attrsResult.value === 'object' &&
    !Array.isArray(attrsResult.value)
  const invValid = invResult.ok && Array.isArray(invResult.value)

  // Mode switching
  const switchToJson = () => {
    setSwitchError('')
    setAttributesText(JSON.stringify(rowsToAttrs(attrRows), null, 2))
    setInventoryText(JSON.stringify(listToInventory(inventoryList), null, 2))
    setEditorMode('json')
  }

  const switchToForm = () => {
    setSwitchError('')
    const ar = tryParseJSON(attributesText)
    const ir = tryParseJSON(inventoryText)
    if (
      !ar.ok ||
      typeof ar.value !== 'object' ||
      Array.isArray(ar.value) ||
      ar.value === null
    ) {
      setSwitchError('屬性 JSON 格式不正確，無法切換至表單模式')
      return
    }
    if (!ir.ok || !Array.isArray(ir.value)) {
      setSwitchError('物品欄 JSON 格式不正確，無法切換至表單模式')
      return
    }
    setAttrRows(attrsToRows(ar.value as Record<string, unknown>))
    setInventoryList(inventoryToList(ir.value as unknown[]))
    setEditorMode('form')
  }

  // Attribute row helpers
  const updateAttrRow = (index: number, patch: Partial<AttrRow>) => {
    const next = [...attrRows]
    next[index] = { ...next[index], ...patch }
    setAttrRows(next)
  }

  const removeAttrRow = (index: number) => {
    setAttrRows(attrRows.filter((_, i) => i !== index))
  }

  const addAttrRow = () => {
    setAttrRows([...attrRows, { key: '', value: '' }])
  }

  // Inventory list helpers
  const updateInventoryItem = (index: number, value: string) => {
    const next = [...inventoryList]
    next[index] = value
    setInventoryList(next)
  }

  const removeInventoryItem = (index: number) => {
    setInventoryList(inventoryList.filter((_, i) => i !== index))
  }

  const addInventoryItem = () => {
    setInventoryList([...inventoryList, ''])
  }

  async function handleSubmit() {
    if (!name.trim()) {
      setError('名稱為必填')
      return
    }

    let attributes: Record<string, unknown>
    let inventory: unknown[]

    if (editorMode === 'form') {
      attributes = rowsToAttrs(attrRows)
      inventory = listToInventory(inventoryList)
    } else {
      if (!attrsValid) {
        setError('屬性必須是有效的 JSON 物件')
        return
      }
      if (!invValid) {
        setError('物品欄必須是有效的 JSON 陣列')
        return
      }
      attributes = (attrsResult as { ok: true; value: unknown })
        .value as Record<string, unknown>
      inventory = (invResult as { ok: true; value: unknown })
        .value as unknown[]
    }

    setError('')
    setLoading(true)

    const data = {
      name: name.trim(),
      attributes,
      inventory,
      notes: notes.trim(),
    }

    try {
      if (isEdit) {
        await updateCharacter(character.id, data)
      } else {
        await createCharacter(data)
      }
      onSaved()
      onClose()
    } catch (err) {
      if (err instanceof ApiClientError) {
        setError(err.body.message)
      } else {
        setError('發生未預期的錯誤')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-[#0F0F0FCC]"
      onClick={onClose}
    >
      <div
        className="flex w-full max-w-130 max-h-[85vh] flex-col gap-5 rounded-xl bg-bg-card p-8 overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
      >
        <h2 className="font-display text-xl font-semibold text-text-primary">
          {isEdit ? '編輯角色' : '建立角色'}
        </h2>

        {/* Name */}
        <div className="flex flex-col gap-1">
          <label htmlFor="char-name" className="text-sm text-text-secondary">
            名稱
          </label>
          <Input
            id="char-name"
            placeholder="角色名稱"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </div>

        {/* Notes */}
        <div className="flex flex-col gap-1">
          <label htmlFor="char-notes" className="text-sm text-text-secondary">
            筆記
          </label>
          <Input
            id="char-notes"
            placeholder="選填筆記"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
          />
        </div>

        {/* Mode toggle */}
        <div className="flex flex-col gap-2">
          <div className="flex border-b border-border">
            <button
              type="button"
              className={cn(
                'px-4 py-2 text-xs font-medium transition-colors',
                editorMode === 'form'
                  ? 'border-b-2 border-gold text-gold'
                  : 'text-text-tertiary hover:text-text-secondary',
              )}
              onClick={() =>
                editorMode === 'json' ? switchToForm() : undefined
              }
            >
              表單模式
            </button>
            <button
              type="button"
              className={cn(
                'px-4 py-2 text-xs font-medium transition-colors',
                editorMode === 'json'
                  ? 'border-b-2 border-gold text-gold'
                  : 'text-text-tertiary hover:text-text-secondary',
              )}
              onClick={() =>
                editorMode === 'form' ? switchToJson() : undefined
              }
            >
              JSON 模式
            </button>
          </div>

          {switchError && (
            <p className="text-xs text-error">{switchError}</p>
          )}
        </div>

        {/* Editor content */}
        {editorMode === 'form' ? (
          <div className="flex flex-col gap-5">
            {/* Attributes */}
            <div className="flex flex-col gap-2">
              <div className="flex items-center justify-between">
                <span className="text-sm text-text-secondary">屬性</span>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={addAttrRow}
                  type="button"
                >
                  + 新增屬性
                </Button>
              </div>

              {attrRows.length === 0 && (
                <p className="text-xs text-text-tertiary">尚未新增屬性</p>
              )}

              {attrRows.map((row, i) => (
                <div
                  key={i}
                  className="flex items-end gap-2 rounded-lg border border-border bg-[#1A1A1A] px-3 py-2.5"
                >
                  <label className="flex flex-1 flex-col gap-1">
                    <span className="text-xs text-text-tertiary">Key</span>
                    <Input
                      value={row.key}
                      onChange={(e) =>
                        updateAttrRow(i, { key: e.target.value })
                      }
                      placeholder="屬性名稱"
                    />
                  </label>
                  <label className="flex flex-1 flex-col gap-1">
                    <span className="text-xs text-text-tertiary">Value</span>
                    <Input
                      value={row.value}
                      onChange={(e) =>
                        updateAttrRow(i, { value: e.target.value })
                      }
                      placeholder="屬性值"
                    />
                  </label>
                  <button
                    type="button"
                    onClick={() => removeAttrRow(i)}
                    className="shrink-0 pb-2.5 text-sm text-text-tertiary transition-colors hover:text-error"
                  >
                    刪除
                  </button>
                </div>
              ))}
            </div>

            {/* Inventory */}
            <div className="flex flex-col gap-2">
              <div className="flex items-center justify-between">
                <span className="text-sm text-text-secondary">物品欄</span>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={addInventoryItem}
                  type="button"
                >
                  + 新增物品
                </Button>
              </div>

              {inventoryList.length === 0 && (
                <p className="text-xs text-text-tertiary">尚未新增物品</p>
              )}

              {inventoryList.map((item, i) => (
                <div
                  key={i}
                  className="flex items-center gap-2 rounded-lg border border-border bg-[#1A1A1A] px-3 py-2.5"
                >
                  <Input
                    value={item}
                    onChange={(e) => updateInventoryItem(i, e.target.value)}
                    placeholder="物品名稱或 JSON"
                    className="flex-1"
                  />
                  <button
                    type="button"
                    onClick={() => removeInventoryItem(i)}
                    className="shrink-0 text-sm text-text-tertiary transition-colors hover:text-error"
                  >
                    刪除
                  </button>
                </div>
              ))}
            </div>
          </div>
        ) : (
          <div className="flex flex-col gap-4">
            {/* Attributes JSON */}
            <div className="flex flex-col gap-1">
              <label
                htmlFor="char-attrs"
                className="text-sm text-text-secondary"
              >
                屬性 (JSON)
              </label>
              <Textarea
                id="char-attrs"
                rows={4}
                className="font-mono"
                value={attributesText}
                onChange={(e) => setAttributesText(e.target.value)}
              />
              <span
                className={`text-xs ${attrsValid ? 'text-green-500' : 'text-error'}`}
              >
                {attrsValid ? '\u2713 JSON 格式正確' : '\u2717 無效的 JSON 物件'}
              </span>
            </div>

            {/* Inventory JSON */}
            <div className="flex flex-col gap-1">
              <label
                htmlFor="char-inv"
                className="text-sm text-text-secondary"
              >
                物品欄 (JSON)
              </label>
              <Textarea
                id="char-inv"
                rows={3}
                className="font-mono"
                value={inventoryText}
                onChange={(e) => setInventoryText(e.target.value)}
              />
              <span
                className={`text-xs ${invValid ? 'text-green-500' : 'text-error'}`}
              >
                {invValid ? '\u2713 JSON 格式正確' : '\u2717 無效的 JSON 陣列'}
              </span>
            </div>
          </div>
        )}

        {error && <p className="text-xs text-error">{error}</p>}

        <div className="flex gap-3">
          <Button
            variant="ghost"
            className="flex-1"
            onClick={onClose}
            disabled={loading}
          >
            取消
          </Button>
          <Button className="flex-1" onClick={handleSubmit} loading={loading}>
            {isEdit ? '儲存' : '建立'}
          </Button>
        </div>
      </div>
    </div>
  )
}
