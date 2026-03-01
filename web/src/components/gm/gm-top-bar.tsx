import { useState } from 'react'
import type { SessionStatus } from '../../api/types'
import { Button } from '../ui/button'
import { ConfirmModal } from '../scenario/confirm-modal'

interface GmTopBarProps {
  scenarioTitle: string
  status: SessionStatus
  onPause: () => void
  onResume: () => void
  onEnd: () => void
}

export function GmTopBar({
  scenarioTitle,
  status,
  onPause,
  onResume,
  onEnd,
}: GmTopBarProps) {
  const [showEndConfirm, setShowEndConfirm] = useState(false)

  return (
    <div className="flex h-14 items-center justify-between bg-bg-sidebar px-6">
      <div className="flex items-center gap-3">
        <span className="font-display text-lg font-bold text-gold">TRPG</span>
        <div className="h-5 w-px bg-border" />
        <span className="text-sm text-text-secondary">{scenarioTitle}</span>
      </div>

      <div className="flex items-center gap-2">
        {status === 'active' && (
          <Button variant="secondary" size="sm" onClick={onPause}>
            Pause
          </Button>
        )}
        {status === 'paused' && (
          <Button variant="secondary" size="sm" onClick={onResume}>
            Resume
          </Button>
        )}
        {(status === 'active' || status === 'paused') && (
          <Button
            variant="ghost"
            size="sm"
            className="text-error"
            onClick={() => setShowEndConfirm(true)}
          >
            End Game
          </Button>
        )}
      </div>

      <ConfirmModal
        open={showEndConfirm}
        onClose={() => setShowEndConfirm(false)}
        onConfirm={() => {
          setShowEndConfirm(false)
          onEnd()
        }}
        title="End Game?"
        description="This will permanently end the game session. All players will be disconnected. This action cannot be undone."
        confirmLabel="End Game"
        confirmClassName="bg-error hover:bg-error/80 text-white"
      />
    </div>
  )
}
