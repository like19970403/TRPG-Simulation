import { useRef, useState } from 'react'
import { uploadImage } from '../../api/upload'

interface ImageUploadProps {
  /** Current image URL (or undefined). */
  value?: string
  /** Called with the new URL after a successful upload, or undefined to clear. */
  onChange: (url: string | undefined) => void
  /** Label text. */
  label?: string
  /** CSS class for the preview image. */
  previewClass?: string
}

export function ImageUpload({
  value,
  onChange,
  label = '圖片（選填）',
  previewClass = 'h-20 w-20 rounded-lg object-cover',
}: ImageUploadProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState('')

  async function handleFile(file: File) {
    setError('')
    setUploading(true)
    try {
      const result = await uploadImage(file)
      onChange(result.url)
    } catch (err) {
      setError(err instanceof Error ? err.message : '上傳失敗')
    } finally {
      setUploading(false)
    }
  }

  return (
    <div className="flex flex-col gap-1">
      <span className="text-xs font-medium text-text-secondary">{label}</span>
      <div className="flex items-center gap-3">
        {value && (
          <img src={value} alt="preview" className={previewClass} />
        )}
        <div className="flex flex-col gap-1">
          <div className="flex items-center gap-2">
            <button
              type="button"
              disabled={uploading}
              onClick={() => inputRef.current?.click()}
              className="rounded border border-border bg-bg-card px-3 py-1 text-xs text-text-secondary transition-colors hover:bg-bg-secondary disabled:opacity-50"
            >
              {uploading ? '上傳中...' : value ? '更換圖片' : '選擇圖片'}
            </button>
            {value && (
              <button
                type="button"
                onClick={() => onChange(undefined)}
                className="text-xs text-text-tertiary transition-colors hover:text-error"
              >
                移除
              </button>
            )}
          </div>
          <span className="text-[10px] text-text-tertiary">
            JPEG / PNG / GIF / WebP，最大 5MB
          </span>
        </div>
        <input
          ref={inputRef}
          type="file"
          accept="image/jpeg,image/png,image/gif,image/webp"
          className="hidden"
          onChange={(e) => {
            const file = e.target.files?.[0]
            if (file) handleFile(file)
            e.target.value = ''
          }}
        />
      </div>
      {error && <p className="text-xs text-error">{error}</p>}
    </div>
  )
}
