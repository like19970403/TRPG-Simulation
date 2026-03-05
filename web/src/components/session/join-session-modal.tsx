import { useState, useRef } from 'react'
import { useNavigate } from 'react-router'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { joinSession } from '../../api/sessions'
import { ApiClientError } from '../../api/client'
import { useFocusTrap } from '../../hooks/use-focus-trap'

interface JoinSessionModalProps {
  open: boolean
  onClose: () => void
  onJoined: () => void
}

export function JoinSessionModal({ open, onClose, onJoined }: JoinSessionModalProps) {
  const navigate = useNavigate()
  const dialogRef = useRef<HTMLDivElement>(null)
  useFocusTrap(dialogRef, open)
  const [code, setCode] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  if (!open) return null

  async function handleJoin() {
    const trimmed = code.trim()
    if (!trimmed) {
      setError('請輸入邀請碼')
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
        setError('加入場次失敗')
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
        ref={dialogRef}
        className="flex w-full max-w-[400px] flex-col gap-5 rounded-xl bg-bg-card p-8"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
      >
        <h2 className="font-display text-xl font-semibold text-text-primary">
          加入場次
        </h2>
        <p className="text-sm text-text-secondary">
          輸入 GM 分享的邀請碼來加入遊戲場次。
        </p>

        <Input
          placeholder="輸入邀請碼"
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
            取消
          </Button>
          <Button className="flex-1" onClick={handleJoin} loading={loading}>
            加入
          </Button>
        </div>
      </div>
    </div>
  )
}
