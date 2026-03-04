import { useEffect } from 'react'

interface ShortcutDef {
  /** Key code (e.g. 'Digit1', 'KeyD', 'Enter', 'Escape') */
  key: string
  ctrl?: boolean
  handler: () => void
}

/**
 * Register keyboard shortcuts that are ignored when an input/textarea is focused.
 * @param shortcuts Array of shortcut definitions
 * @param deps React dependency array for re-registration
 */
export function useKeyboardShortcuts(shortcuts: ShortcutDef[], deps: unknown[] = []) {
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      // Skip when typing in inputs
      const tag = (e.target as HTMLElement)?.tagName
      if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return
      // Skip when IME is composing
      if (e.isComposing) return

      for (const s of shortcuts) {
        if (s.key === e.code && !!s.ctrl === (e.ctrlKey || e.metaKey)) {
          e.preventDefault()
          s.handler()
          return
        }
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps)
}
