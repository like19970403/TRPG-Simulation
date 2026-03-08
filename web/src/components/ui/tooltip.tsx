import { useState, useRef, useEffect, type ReactNode } from 'react'

interface TooltipProps {
  content: string
  children: ReactNode
}

export function Tooltip({ content, children }: TooltipProps) {
  const [show, setShow] = useState(false)
  const [style, setStyle] = useState<React.CSSProperties>({})
  const triggerRef = useRef<HTMLSpanElement>(null)

  useEffect(() => {
    if (show && triggerRef.current) {
      const rect = triggerRef.current.getBoundingClientRect()
      const above = rect.top >= 80
      setStyle({
        position: 'fixed',
        left: Math.max(8, rect.left + rect.width / 2 - 140),
        ...(above
          ? { top: rect.top - 8, transform: 'translateY(-100%)' }
          : { top: rect.bottom + 8 }),
        zIndex: 9999,
      })
    }
  }, [show])

  return (
    <span
      ref={triggerRef}
      className="inline-flex"
      onMouseEnter={() => setShow(true)}
      onMouseLeave={() => setShow(false)}
    >
      {children}
      {show && (
        <span
          className="w-max max-w-xs whitespace-normal rounded-lg bg-bg-card px-3 py-2 text-xs leading-relaxed text-text-secondary shadow-lg ring-1 ring-border"
          style={style}
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
