import { useState, useEffect, useCallback } from 'react'

interface NotesPanelProps {
  sessionId: string
  /** Optional CSS class for the outer container */
  className?: string
}

const STORAGE_PREFIX = 'trpg-notes-'

export function NotesPanel({ sessionId, className }: NotesPanelProps) {
  const storageKey = STORAGE_PREFIX + sessionId

  const [text, setText] = useState(() => {
    try {
      return localStorage.getItem(storageKey) ?? ''
    } catch {
      return ''
    }
  })

  // Auto-save to localStorage on change (debounced)
  useEffect(() => {
    const timer = setTimeout(() => {
      try {
        localStorage.setItem(storageKey, text)
      } catch {
        // Ignore storage errors
      }
    }, 500)
    return () => clearTimeout(timer)
  }, [text, storageKey])

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      setText(e.target.value)
    },
    [],
  )

  return (
    <div className={className ?? 'flex flex-1 flex-col p-4'}>
      <textarea
        className="flex-1 resize-none rounded-lg border border-border bg-bg-card p-3 text-sm text-text-primary placeholder:text-text-tertiary focus:border-gold/50 focus:outline-none"
        placeholder="在這裡寫筆記..."
        value={text}
        onChange={handleChange}
      />
    </div>
  )
}
