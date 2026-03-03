import { useRef, useState } from 'react'
import { useGameStore } from '../../stores/game-store'
import { uploadImage } from '../../api/upload'
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
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState('')
  const fileRef = useRef<HTMLInputElement>(null)

  async function handleFileUpload(file: File) {
    setError('')
    setUploading(true)
    try {
      const result = await uploadImage(file)
      setImageUrl(result.url)
    } catch (err) {
      setError(err instanceof Error ? err.message : '上傳失敗')
    } finally {
      setUploading(false)
    }
  }

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
        <div className="flex items-center gap-1">
          <Input
            placeholder="圖片網址"
            value={imageUrl}
            onChange={(e) => setImageUrl(e.target.value)}
            className="w-36"
          />
          <button
            type="button"
            disabled={uploading}
            onClick={() => fileRef.current?.click()}
            className="shrink-0 rounded border border-border bg-bg-card px-2 py-1.5 text-xs text-text-secondary transition-colors hover:bg-bg-secondary disabled:opacity-50"
            title="上傳圖片"
          >
            {uploading ? '...' : '📎'}
          </button>
          <input
            ref={fileRef}
            type="file"
            accept="image/jpeg,image/png,image/gif,image/webp"
            className="hidden"
            onChange={(e) => {
              const file = e.target.files?.[0]
              if (file) handleFileUpload(file)
              e.target.value = ''
            }}
          />
        </div>
        <Button variant="primary" size="sm" onClick={handleSend}>
          發送
        </Button>
      </div>
      {imageUrl && (
        <div className="flex items-center gap-2">
          <img
            src={imageUrl}
            alt="preview"
            className="h-10 w-10 rounded object-cover"
          />
          <button
            type="button"
            onClick={() => setImageUrl('')}
            className="text-xs text-text-tertiary hover:text-error"
          >
            移除圖片
          </button>
        </div>
      )}
      {error && <p className="text-xs text-error">{error}</p>}
    </div>
  )
}
