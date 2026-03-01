import { useState } from 'react'
import { useNavigate } from 'react-router'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { joinSession } from '../../api/sessions'
import { ApiClientError } from '../../api/client'

interface JoinSessionModalProps {
  open: boolean
  onClose: () => void
  onJoined: () => void
}

export function JoinSessionModal({ open, onClose, onJoined }: JoinSessionModalProps) {
  const navigate = useNavigate()
  const [code, setCode] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  if (!open) return null

  async function handleJoin() {
    const trimmed = code.trim()
    if (!trimmed) {
      setError('Enter an invite code')
      return
    }

    setError('')
    setLoading(true)
    try {
      const session = await joinSession({ inviteCode: trimmed })
      onJoined()
      onClose()
      navigate(`/sessions/${session.id}/lobby`)
    } catch (err) {
      if (err instanceof ApiClientError) {
        setError(err.body.message)
      } else {
        setError('Failed to join session')
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
        className="flex w-full max-w-[400px] flex-col gap-5 rounded-xl bg-bg-card p-8"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
      >
        <h2 className="font-display text-xl font-semibold text-text-primary">
          Join Session
        </h2>
        <p className="text-sm text-text-secondary">
          Enter the invite code shared by the GM to join a game session.
        </p>

        <Input
          placeholder="Enter invite code"
          value={code}
          onChange={(e) => setCode(e.target.value.toUpperCase())}
          error={!!error}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault()
              handleJoin()
            }
          }}
        />
        {error && <p className="text-xs text-error">{error}</p>}

        <div className="flex gap-3">
          <Button variant="ghost" className="flex-1" onClick={onClose} disabled={loading}>
            Cancel
          </Button>
          <Button className="flex-1" onClick={handleJoin} loading={loading}>
            Join
          </Button>
        </div>
      </div>
    </div>
  )
}
