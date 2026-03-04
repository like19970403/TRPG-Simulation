import { useState, useCallback } from 'react'
import { Button } from './button'
import { Input } from './input'
import type { SendAction } from '../../hooks/use-game-socket'

const DICE_REGEX = /^\d*d\d+([+-]\d+)?$/
const STORAGE_KEY = 'trpg-dice-cache'
const MAX_CACHE = 8

interface DiceRollerProps {
  sendAction: SendAction
  /** Show purpose input field (GM only) */
  showPurpose?: boolean
}

function loadCache(): string[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as unknown
    if (Array.isArray(parsed)) return parsed.filter((v): v is string => typeof v === 'string')
  } catch {
    // ignore
  }
  return []
}

function saveCache(formulas: string[]) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(formulas))
  } catch {
    // ignore
  }
}

function addToCache(formula: string): string[] {
  const existing = loadCache().filter((f) => f !== formula)
  const updated = [formula, ...existing].slice(0, MAX_CACHE)
  saveCache(updated)
  return updated
}

export function DiceRoller({ sendAction, showPurpose }: DiceRollerProps) {
  const [formula, setFormula] = useState('')
  const [purpose, setPurpose] = useState('')
  const [error, setError] = useState('')
  const [cache, setCache] = useState(loadCache)

  const doRoll = useCallback(
    (f: string, p?: string) => {
      const payload: Record<string, unknown> = { formula: f }
      if (p?.trim()) {
        payload.purpose = p.trim()
      }
      sendAction('dice_roll', payload)
      setCache(addToCache(f))
    },
    [sendAction],
  )

  function handleRoll() {
    const trimmed = formula.trim()
    if (!trimmed) {
      setError('請輸入骰子公式')
      return
    }
    if (!DICE_REGEX.test(trimmed)) {
      setError('無效的公式（例如 2d6、d20+5）')
      return
    }
    setError('')
    doRoll(trimmed, purpose)
    setFormula('')
    setPurpose('')
  }

  function handleQuickRoll(f: string) {
    setError('')
    doRoll(f, purpose)
    if (showPurpose) setPurpose('')
  }

  return (
    <div>
      {/* Quick roll buttons from cache */}
      {cache.length > 0 && (
        <div className="mb-2 flex flex-wrap gap-1">
          {cache.map((f) => (
            <button
              key={f}
              className="rounded border border-border bg-bg-card px-2 py-0.5 font-mono text-xs text-text-secondary transition-colors hover:border-gold/50 hover:text-gold"
              onClick={() => handleQuickRoll(f)}
            >
              {f}
            </button>
          ))}
        </div>
      )}

      {/* Input row */}
      <div className="flex gap-2">
        <div className="w-32">
          <Input
            placeholder="2d6+3"
            value={formula}
            onChange={(e) => setFormula(e.target.value)}
            error={!!error}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault()
                handleRoll()
              }
            }}
          />
        </div>
        {showPurpose && (
          <div className="flex-1">
            <Input
              placeholder="用途（選填）"
              value={purpose}
              onChange={(e) => setPurpose(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault()
                  handleRoll()
                }
              }}
            />
          </div>
        )}
        <Button variant="primary" size="sm" onClick={handleRoll}>
          擲骰
        </Button>
      </div>
      {error && <p className="mt-1 text-xs text-error">{error}</p>}
    </div>
  )
}
