import { useState, useEffect } from 'react'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { createCharacter, updateCharacter } from '../../api/characters'
import { ApiClientError } from '../../api/client'
import type { CharacterResponse } from '../../api/types'

interface CharacterFormModalProps {
  open: boolean
  onClose: () => void
  onSaved: () => void
  character?: CharacterResponse | null
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
  const [attributesText, setAttributesText] = useState('{}')
  const [inventoryText, setInventoryText] = useState('[]')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (open && character) {
      setName(character.name)
      setNotes(character.notes)
      setAttributesText(JSON.stringify(character.attributes, null, 2))
      setInventoryText(JSON.stringify(character.inventory, null, 2))
      setError('')
    } else if (open) {
      setName('')
      setNotes('')
      setAttributesText('{}')
      setInventoryText('[]')
      setError('')
    }
  }, [open, character])

  if (!open) return null

  const attrsResult = tryParseJSON(attributesText)
  const invResult = tryParseJSON(inventoryText)
  const attrsValid = attrsResult.ok && typeof attrsResult.value === 'object' && !Array.isArray(attrsResult.value)
  const invValid = invResult.ok && Array.isArray(invResult.value)

  async function handleSubmit() {
    if (!name.trim()) {
      setError('Name is required')
      return
    }
    if (!attrsValid) {
      setError('Attributes must be a valid JSON object')
      return
    }
    if (!invValid) {
      setError('Inventory must be a valid JSON array')
      return
    }

    setError('')
    setLoading(true)

    const data = {
      name: name.trim(),
      attributes: (attrsResult as { ok: true; value: unknown }).value as Record<string, unknown>,
      inventory: (invResult as { ok: true; value: unknown }).value as unknown[],
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
        setError('An unexpected error occurred')
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
        className="flex w-full max-w-[480px] flex-col gap-5 rounded-xl bg-bg-card p-8"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
      >
        <h2 className="font-display text-xl font-semibold text-text-primary">
          {isEdit ? 'Edit Character' : 'Create Character'}
        </h2>

        <div className="flex flex-col gap-1">
          <label htmlFor="char-name" className="text-sm text-text-secondary">
            Name
          </label>
          <Input
            id="char-name"
            placeholder="Character name"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </div>

        <div className="flex flex-col gap-1">
          <label htmlFor="char-notes" className="text-sm text-text-secondary">
            Notes
          </label>
          <Input
            id="char-notes"
            placeholder="Optional notes"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
          />
        </div>

        <div className="flex flex-col gap-1">
          <label htmlFor="char-attrs" className="text-sm text-text-secondary">
            Attributes (JSON)
          </label>
          <textarea
            id="char-attrs"
            className="w-full rounded-lg border border-border bg-bg-input px-3 py-2.5 font-mono text-sm text-text-primary outline-none transition-colors placeholder:text-text-tertiary focus:border-border-focus"
            rows={3}
            value={attributesText}
            onChange={(e) => setAttributesText(e.target.value)}
          />
          <span className={`text-xs ${attrsValid ? 'text-green-500' : 'text-error'}`}>
            {attrsValid ? '\u2713 Valid JSON' : '\u2717 Invalid JSON object'}
          </span>
        </div>

        <div className="flex flex-col gap-1">
          <label htmlFor="char-inv" className="text-sm text-text-secondary">
            Inventory (JSON)
          </label>
          <textarea
            id="char-inv"
            className="w-full rounded-lg border border-border bg-bg-input px-3 py-2.5 font-mono text-sm text-text-primary outline-none transition-colors placeholder:text-text-tertiary focus:border-border-focus"
            rows={2}
            value={inventoryText}
            onChange={(e) => setInventoryText(e.target.value)}
          />
          <span className={`text-xs ${invValid ? 'text-green-500' : 'text-error'}`}>
            {invValid ? '\u2713 Valid JSON' : '\u2717 Invalid JSON array'}
          </span>
        </div>

        {error && <p className="text-xs text-error">{error}</p>}

        <div className="flex gap-3">
          <Button
            variant="ghost"
            className="flex-1"
            onClick={onClose}
            disabled={loading}
          >
            Cancel
          </Button>
          <Button className="flex-1" onClick={handleSubmit} loading={loading}>
            {isEdit ? 'Save' : 'Create'}
          </Button>
        </div>
      </div>
    </div>
  )
}
