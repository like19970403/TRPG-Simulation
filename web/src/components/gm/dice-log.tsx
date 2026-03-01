import { useState } from 'react'
import { useGameStore } from '../../stores/game-store'
import { Button } from '../ui/button'
import { Input } from '../ui/input'

const DICE_REGEX = /^\d*d\d+([+-]\d+)?$/

interface DiceLogProps {
  sendAction: (type: string, payload: unknown) => void
}

export function DiceLog({ sendAction }: DiceLogProps) {
  const diceHistory = useGameStore(
    (s) => s.gameState?.dice_history ?? [],
  )
  const [formula, setFormula] = useState('')
  const [purpose, setPurpose] = useState('')
  const [error, setError] = useState('')

  function handleRoll() {
    const trimmed = formula.trim()
    if (!trimmed) {
      setError('Enter a dice formula')
      return
    }
    if (!DICE_REGEX.test(trimmed)) {
      setError('Invalid formula (e.g. 2d6, d20+5)')
      return
    }

    setError('')
    const payload: Record<string, unknown> = { formula: trimmed }
    if (purpose.trim()) {
      payload.purpose = purpose.trim()
    }
    sendAction('dice_roll', payload)
    setFormula('')
    setPurpose('')
  }

  return (
    <div className="flex flex-1 flex-col p-4">
      {/* Roll input */}
      <div className="mb-3 flex gap-2">
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
        <div className="flex-1">
          <Input
            placeholder="Purpose (optional)"
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
        <Button variant="primary" size="sm" onClick={handleRoll}>
          Roll
        </Button>
      </div>
      {error && <p className="mb-2 text-xs text-error">{error}</p>}

      {/* Dice history */}
      <div className="flex-1 overflow-y-auto">
        {diceHistory.length === 0 ? (
          <p className="text-xs text-text-tertiary">No dice rolled yet</p>
        ) : (
          <div className="flex flex-col gap-1">
            {[...diceHistory].reverse().map((dr, i) => (
              <div
                key={`dice-${i}`}
                className="flex items-center gap-2 text-xs"
              >
                <span className="font-mono font-medium text-gold">
                  {dr.formula}
                </span>
                <span className="text-text-tertiary">
                  [{dr.results.join(', ')}]
                  {dr.modifier !== 0 &&
                    (dr.modifier > 0
                      ? `+${dr.modifier}`
                      : `${dr.modifier}`)}
                </span>
                <span className="font-medium text-text-primary">
                  = {dr.total}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
