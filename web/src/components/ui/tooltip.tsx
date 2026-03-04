import { useState, useRef, useEffect, type ReactNode } from 'react'

interface TooltipProps {
  content: string
  children: ReactNode
}

export function Tooltip({ content, children }: TooltipProps) {
  const [show, setShow] = useState(false)
  const [pos, setPos] = useState<'top' | 'bottom'>('top')
  const triggerRef = useRef<HTMLSpanElement>(null)

  useEffect(() => {
    if (show && triggerRef.current) {
      const rect = triggerRef.current.getBoundingClientRect()
      setPos(rect.top < 80 ? 'bottom' : 'top')
    }
  }, [show])

  return (
    <span
      ref={triggerRef}
      className="relative inline-flex"
      onMouseEnter={() => setShow(true)}
      onMouseLeave={() => setShow(false)}
    >
      {children}
      {show && (
        <span
          className={`absolute left-1/2 z-50 max-w-52 -translate-x-1/2 whitespace-normal rounded-lg bg-bg-card px-3 py-2 text-xs leading-relaxed text-text-secondary shadow-lg ring-1 ring-border ${
            pos === 'top' ? 'bottom-full mb-2' : 'top-full mt-2'
          }`}
        >
          {content}
        </span>
      )}
    </span>
  )
}

export function HelpIcon({ tip }: { tip: string }) {
  return (
    <Tooltip content={tip}>
      <span className="inline-flex h-4 w-4 cursor-help items-center justify-center rounded-full border border-border text-[10px] text-text-tertiary hover:border-gold/50 hover:text-gold">
        ?
      </span>
    </Tooltip>
  )
}
