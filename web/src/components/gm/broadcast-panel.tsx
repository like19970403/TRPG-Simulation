import { useState } from 'react'
import { useGameStore } from '../../stores/game-store'
import { Button } from '../ui/button'
import { Input } from '../ui/input'

interface BroadcastPanelProps {
  sendAction: (type: string, payload: unknown) => void
}

const EMPTY_PLAYERS: Record<string, unknown> = {}

export function BroadcastPanel({ sendAction }: BroadcastPanelProps) {
  const players = useGameStore(
    (s) => s.gameState?.players ?? EMPTY_PLAYERS,
  )
  const [content, setContent] = useState('')
  const [imageUrl, setImageUrl] = useState('')
  const [error, setError] = useState('')

  function handleSend() {
    if (!content.trim() && !imageUrl.trim()) {
      setError('請輸入訊息或圖片網址')
      return
    }

    setError('')

    const payload: Record<string, unknown> = {
      player_ids: Object.keys(players),
    }
    if (content.trim()) payload.content = content.trim()
    if (imageUrl.trim()) payload.image_url = imageUrl.trim()

    sendAction('gm_broadcast', payload)
    setContent('')
    setImageUrl('')
  }

  return (
    <div className="flex flex-1 flex-col gap-3 p-4">
      <div className="flex gap-3">
        <div className="flex-1">
          <Input
            placeholder="給玩家的訊息..."
            value={content}
            onChange={(e) => setContent(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault()
                handleSend()
              }
            }}
          />
        </div>
        <div className="w-48">
          <Input
            placeholder="圖片網址（選填）"
            value={imageUrl}
            onChange={(e) => setImageUrl(e.target.value)}
          />
        </div>
        <Button variant="primary" size="sm" onClick={handleSend}>
          發送
        </Button>
      </div>
      {error && <p className="text-xs text-error">{error}</p>}
    </div>
  )
}
