import { useState, useCallback } from 'react'
import { cn } from '../../lib/cn'

interface ContentEditorProps {
  value: string
  onChange: (value: string) => void
  error?: string
  readOnly?: boolean
}

export function ContentEditor({
  value,
  onChange,
  error,
  readOnly = false,
}: ContentEditorProps) {
  const [jsonError, setJsonError] = useState<string | null>(null)

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      const newValue = e.target.value
      onChange(newValue)

      if (newValue.trim() === '') {
        setJsonError(null)
        return
      }

      try {
        JSON.parse(newValue)
        setJsonError(null)
      } catch (err) {
        setJsonError(err instanceof Error ? err.message : 'Invalid JSON')
      }
    },
    [onChange],
  )

  const displayError = error ?? jsonError

  return (
    <div className="flex flex-col gap-1.5">
      <textarea
        value={value}
        onChange={handleChange}
        readOnly={readOnly}
        className={cn(
          'min-h-[280px] w-full resize-y rounded-md bg-[#1A1A1A] p-4 font-mono text-xs text-text-primary',
          'border outline-none transition-colors',
          'placeholder:text-text-tertiary',
          displayError
            ? 'border-error focus:border-error'
            : 'border-border focus:border-border-focus',
          readOnly && 'cursor-default',
        )}
        placeholder='{"startScene": "entrance", "scenes": [...]}'
        spellCheck={false}
      />
      {value.trim() !== '' && !displayError && (
        <span className="text-xs text-success">Valid JSON</span>
      )}
      {displayError && <p className="text-xs text-error">{displayError}</p>}
    </div>
  )
}
